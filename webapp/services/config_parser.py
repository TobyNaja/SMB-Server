import os
import re
import shlex
import logging
from typing import Dict, List, Optional

logger = logging.getLogger(__name__)

SYSTEM_SECTIONS = {'global', 'homes', 'printers', 'print$', 'netlogon', 'sysvol'}
USER_LIST_FIELDS = {'valid users', 'invalid users', 'read list', 'write list', 'admin users'}

_INVALID_USER_CHARS = re.compile(r'[^\w\\@\s.\-]')  # กัน ×, emoji, etc.

def _sanitize_users(users: List[str]) -> List[str]:
    """กรอง characters ที่ไม่ถูกต้องออกจาก username ก่อนบันทึก"""
    result = []
    for u in users:
        clean = _INVALID_USER_CHARS.sub('', u).strip()
        if not clean:
            logger.warning(f"User '{u}' → empty after sanitize, skipped")
            continue
        if clean != u:
            logger.warning(f"Sanitized: '{u}' → '{clean}'")
        result.append(clean)
    return result


# ════════════════════════════════════════════════
#  User List Helpers
# ════════════════════════════════════════════════

def _parse_user_list(raw: str) -> List[str]:
    """
    Parse user list จาก smb.conf ให้รองรับ quoted strings และ AD Users (Backslash)
    """
    if not raw:
        return []
    
    import re
    # ใช้ Regex ดึงคำแยกด้วยช่องว่าง แต่ถ้ามีเครื่องหมายคำพูดครอบ (รวมถึงมี @ นำหน้า) ให้มัดรวมกันไว้
    tokens = re.findall(r'@?"[^"]+"|\S+', raw)
    
    # ถอดเครื่องหมายคำพูด " ออก โดยที่เครื่องหมาย \ ยังอยู่ครบถ้วน
    return [t.replace('"', '') for t in tokens if t]

def _format_user(user: str) -> str:
    """
    Format user/group entry ให้ถูกต้องตาม smb.conf syntax

    'localadmin'       → 'localadmin'
    'IT\\john'           → '"IT\\john"'            (quote เพราะมี backslash)
    '@IT\\Domain Users'  → '@"IT\\Domain Users"'   (group + space)
    """
    # แยก @ prefix (group indicator) ออกก่อน
    if user.startswith('@'):
        prefix = '@'
        name = user[1:]
    else:
        prefix = ''
        name = user

    # Quote ถ้ามี space หรือ backslash (AD users/groups)
    if ' ' in name or '\\' in name:
        return f'{prefix}"{name}"'

    return f'{prefix}{name}'


def _format_user_list(users: List[str]) -> str:
    """แปลง list of users → smb.conf string"""
    return ' '.join(_format_user(u) for u in users if u)


# ════════════════════════════════════════════════
#  Parser Class
# ════════════════════════════════════════════════

class SmbConfParser:
    def __init__(
        self,
        shares_path: str = "/etc/samba/shares.conf",   # ✅ Web UI เขียนที่นี่เท่านั้น
        global_path: str = "/etc/samba/smb.conf",       # ✅ อ่านอย่างเดียว (จาก template)
    ):
        self.config_path = shares_path
        self.global_path = global_path
        self.sections: Dict[str, Dict[str, str]] = {}
        self._parse()

    # ────────────────────────────────────────────
    #  Internal: Parse
    # ────────────────────────────────────────────

    def _parse_file(self, path: str) -> Dict[str, Dict[str, str]]:
        """อ่านไฟล์ smb config ใดก็ได้ → dict"""
        result: Dict[str, Dict[str, str]] = {}
        if not os.path.exists(path):
            return result

        section = None
        with open(path, 'r') as f:
            for line in f:
                line = line.strip()
                if not line or line.startswith(('#', ';')):
                    continue
                if line.startswith('[') and line.endswith(']'):
                    section = line[1:-1].strip()
                    result[section] = {}
                elif section and '=' in line:
                    key, _, val = line.partition('=')
                    result[section][key.strip()] = val.strip()
        return result

    def _parse(self):
        """
        อ่าน shares จาก shares.conf เท่านั้น
        ไม่แตะ smb.conf ([global] ปลอดภัย)
        """
        self.sections = {}

        if not os.path.exists(self.config_path):
            logger.info(f"shares.conf not found at {self.config_path}, starting empty")
            return

        parsed = self._parse_file(self.config_path)

        # โหลดเฉพาะ share sections (ข้าม system sections)
        self.sections = {
            k: v for k, v in parsed.items()
            if k not in SYSTEM_SECTIONS
        }
        logger.info(f"Loaded {len(self.sections)} shares from {self.config_path}")

    # ────────────────────────────────────────────
    #  Internal: Save
    # ────────────────────────────────────────────

    def _save(self):
        """
        เขียนเฉพาะ shares → shares.conf
        ไม่แตะ smb.conf เด็ดขาด
        """
        dir_name = os.path.dirname(self.config_path)
        if dir_name:
            os.makedirs(dir_name, exist_ok=True)

        with open(self.config_path, 'w') as f:
            f.write("# ============================================\n")
            f.write("# SMB Shares - Managed by SMB Manager Web UI\n")
            f.write("# DO NOT EDIT MANUALLY\n")
            f.write("# ============================================\n\n")

            for section, params in self.sections.items():
                if section in SYSTEM_SECTIONS:
                    continue  # Double-check: ห้ามเขียน [global] ลงไปเด็ดขาด

                f.write(f"[{section}]\n")
                for k, v in params.items():
                    f.write(f"    {k} = {v}\n")
                f.write("\n")

        logger.info(f"shares.conf saved: {len(self.sections)} shares")

    # ════════════════════════════════════════════
    #  Global Settings (Read-Only)
    # ════════════════════════════════════════════

    def get_global(self) -> dict:
        """อ่าน global settings จาก smb.conf (read-only ไม่มีการเขียน)"""
        global_conf = self._parse_file(self.global_path)
        g = global_conf.get('global', {})
        return {
            "workgroup":    g.get('workgroup', 'IT'),
            "realm":        g.get('realm', ''),
            "security":     g.get('security', 'ads'),
            "netbios_name": g.get('netbios name', ''),
            "server_string": g.get('server string', 'Samba Server'),
            "abse":         g.get('access based share enum', 'no').lower() == 'yes',
        }

    def set_global_abse(self, enabled: bool) -> bool:
        """Global config อยู่ใน template → ไม่แก้ผ่าน Web UI"""
        logger.warning("Global config is managed by template, cannot modify via Web UI")
        return False

    # ════════════════════════════════════════════
    #  Shares: Read
    # ════════════════════════════════════════════

    def get_shares(self) -> List[str]:
        return [s for s in self.sections if s not in SYSTEM_SECTIONS]

    def get_share(self, name: str) -> Optional[dict]:
        if name not in self.sections:
            return None
        d = self.sections[name]
        return {
            "name":     name,
            "path":     d.get('path', ''),
            "comment":  d.get('comment', ''),
            "browseable":     d.get('browseable', 'yes').lower() == 'yes',
            "read_only":      d.get('read only', 'no').lower() == 'yes',
            "guest_ok":       d.get('guest ok', 'no').lower() == 'yes',
            "abse":           d.get('access based share enum', 'no').lower() == 'yes',
            "valid_users":   _parse_user_list(d.get('valid users', '')),
            "write_list":    _parse_user_list(d.get('write list',  '')),
            "read_list":     _parse_user_list(d.get('read list',   '')),
            "admin_users":   _parse_user_list(d.get('admin users', '')),
            "invalid_users": _parse_user_list(d.get('invalid users', '')),
            "create_mask":    d.get('create mask', '0755'),
            "directory_mask": d.get('directory mask', '0755'),
        }

    def get_all_shares(self) -> List[dict]:
        return [self.get_share(s) for s in self.get_shares()]

    # ════════════════════════════════════════════
    #  Shares: Write
    # ════════════════════════════════════════════

    def create_share(self, name: str, path: str, comment: str = "") -> bool:
        if name in self.sections:
            logger.warning(f"Share '{name}' already exists")
            return False

        self.sections[name] = {
            'comment':                  comment or f'{name} share',
            'path':                     path,
            'browseable':               'yes',
            'read only':                'yes',
            'guest ok':                 'no',
            'access based share enum':  'no',
            'create mask':              '0755',
            'directory mask':           '0755',
            'valid users':              '',  # 🌟 FIXED: บังคับให้เป็นค่าว่างเพื่อทำ Deny All ทันทีที่สร้างโฟลเดอร์
        }
        self._save()
        logger.info(f"Share '{name}' created at {path}")
        return True

    def update_share(self, name: str, updates: dict) -> bool:
        if name not in self.sections:
            return False

        key_map = {
            'read_only': 'read only',
            'guest_ok':  'guest ok',
            'abse':      'access based share enum',
        }
        bool_fields = {'browseable', 'read only', 'guest ok', 'access based share enum'}

        for k, v in updates.items():
            real_key = key_map.get(k, k)
            if real_key in USER_LIST_FIELDS:
                logger.warning(f"Use set_{k}() to update user lists, skipping")
                continue
            if real_key in bool_fields:
                self.sections[name][real_key] = 'yes' if v else 'no'
            else:
                self.sections[name][real_key] = str(v)

        self._save()
        return True

    def delete_share(self, name: str) -> bool:
        if name in self.sections:
            del self.sections[name]
            self._save()
            logger.info(f"Share '{name}' deleted")
            return True
        return False

    # ════════════════════════════════════════════
    #  User List Setters
    # ════════════════════════════════════════════

    def _set_user_list(self, share_name: str, field: str, users: List[str]) -> bool:
        """
        Core setter: เขียน user list พร้อม format ที่ถูกต้อง
        """
        if share_name not in self.sections:
            logger.error(f"Share '{share_name}' not found")
            return False

        users = _sanitize_users(users)

        if users:
            formatted = _format_user_list(users)
            self.sections[share_name][field] = formatted
            logger.info(f"[{share_name}] {field} = {formatted}")
        else:
            # 🌟 FIXED: เปลี่ยนจาก .pop() เป็นกำหนดค่าเป็น string ว่าง เพื่อบังคับให้ Samba ล็อกสิทธิ์แน่นหนาไม่ปล่อยหลวม
            self.sections[share_name][field] = ''
            logger.info(f"[{share_name}] {field} set to empty string → Deny All / Locked")

        self._save()
        return True

    def set_valid_users(self, name: str, users: List[str]) -> bool:
        return self._set_user_list(name, 'valid users', users)

    def set_write_list(self, name: str, users: List[str]) -> bool:
        return self._set_user_list(name, 'write list', users)

    def set_read_list(self, name: str, users: List[str]) -> bool:
        return self._set_user_list(name, 'read list', users)

    def set_admin_users(self, name: str, users: List[str]) -> bool:
        return self._set_user_list(name, 'admin users', users)

    def set_invalid_users(self, name: str, users: List[str]) -> bool:
        return self._set_user_list(name, 'invalid users', users)

    def set_share_abse(self, name: str, enabled: bool) -> bool:
        if name not in self.sections:
            return False
        self.sections[name]['access based share enum'] = 'yes' if enabled else 'no'
        self._save()
        return True
