import os
import logging
from typing import Dict, List, Optional

logger = logging.getLogger(__name__)

SYSTEM_SECTIONS = {'global', 'homes', 'printers', 'print$', 'netlogon', 'sysvol'}

class SmbConfParser:
    def __init__(self, config_path: str):
        self.config_path = config_path
        self.sections: Dict[str, Dict[str, str]] = {}
        self._parse()

    def _parse(self):
        self.sections = {}
        if not os.path.exists(self.config_path):
            logger.warning(f"smb.conf not found: {self.config_path}")
            self.sections = {
                'global': {
                    'workgroup': 'WORKGROUP',
                    'security': 'user',
                    'passdb backend': 'tdbsam'
                }
            }
            return

        with open(self.config_path, 'r') as f:
            section = None
            for line in f:
                line = line.strip()
                if not line or line.startswith(('#', ';')):
                    continue
                if line.startswith('[') and line.endswith(']'):
                    section = line[1:-1].strip()
                    self.sections[section] = {}
                elif section and '=' in line:
                    key, _, val = line.partition('=')
                    self.sections[section][key.strip()] = val.strip()

    def _save(self):
        os.makedirs(os.path.dirname(self.config_path), exist_ok=True)
        with open(self.config_path, 'w') as f:
            f.write("# Auto-managed by SMB Permission Manager\n\n")
            for section, params in self.sections.items():
                f.write(f"[{section}]\n")
                for k, v in params.items():
                    f.write(f"    {k} = {v}\n")
                f.write("\n")
        logger.info("smb.conf saved")

    # ─── Global Settings ──────────────────
    def get_global(self) -> dict:
        """ดู global settings"""
        g = self.sections.get('global', {})
        return {
            "workgroup": g.get('workgroup', 'WORKGROUP'),
            "abse": g.get('access based share enum', 'no').lower() == 'yes',
            "server_string": g.get('server string', 'Samba Server'),
        }

    def set_global_abse(self, enabled: bool) -> bool:
        """เปิด/ปิด ABSE แบบ Global"""
        if 'global' not in self.sections:
            self.sections['global'] = {}
        self.sections['global']['access based share enum'] = 'yes' if enabled else 'no'
        self._save()
        logger.info(f"Global ABSE set to: {enabled}")
        return True

    # ─── Shares ───────────────────────────
    def get_shares(self) -> List[str]:
        return [s for s in self.sections if s not in SYSTEM_SECTIONS]

    def get_share(self, name: str) -> Optional[dict]:
        if name not in self.sections:
            return None
        d = self.sections[name]
        return {
            "name": name,
            "path": d.get('path', ''),
            "comment": d.get('comment', ''),
            "browseable": d.get('browseable', 'yes').lower() == 'yes',
            "read_only": d.get('read only', 'no').lower() == 'yes',
            "guest_ok": d.get('guest ok', 'no').lower() == 'yes',
            "abse": d.get('access based share enum', 'no').lower() == 'yes',
            "valid_users": [u for u in d.get('valid users', '').split() if u],
            "write_list":  [u for u in d.get('write list',  '').split() if u],
            "read_list":   [u for u in d.get('read list',   '').split() if u],
            "admin_users": [u for u in d.get('admin users', '').split() if u],
            "invalid_users": [u for u in d.get('invalid users', '').split() if u],
            "create_mask": d.get('create mask', '0755'),
            "directory_mask": d.get('directory mask', '0755'),
        }

    def get_all_shares(self) -> List[dict]:
        return [self.get_share(s) for s in self.get_shares()]

    def create_share(self, name: str, path: str, comment: str = "") -> bool:
        if name in self.sections:
            return False
        self.sections[name] = {
            'comment': comment or f'{name} share',
            'path': path,
            'browseable': 'yes',
            'read only': 'no',
            'guest ok': 'no',
            'access based share enum': 'no',
            'create mask': '0755',
            'directory mask': '0755',
        }
        self._save()
        return True

    def update_share(self, name: str, updates: dict) -> bool:
        if name not in self.sections:
            return False
        key_map = {
            'read_only':    'read only',
            'guest_ok':     'guest ok',
            'abse':         'access based share enum',
        }
        bool_fields = {'browseable', 'read only', 'guest ok', 'access based share enum'}

        for k, v in updates.items():
            real_key = key_map.get(k, k)
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
            return True
        return False

    def set_valid_users(self, name: str, users: List[str]) -> bool:
        if name not in self.sections:
            return False
        self.sections[name]['valid users'] = ' '.join(users)
        self._save()
        return True

    def set_write_list(self, name: str, users: List[str]) -> bool:
        if name not in self.sections:
            return False
        self.sections[name]['write list'] = ' '.join(users)
        self._save()
        return True

    def set_read_list(self, name: str, users: List[str]) -> bool:
        if name not in self.sections:
            return False
        self.sections[name]['read list'] = ' '.join(users)
        self._save()
        return True

    def set_admin_users(self, name: str, users: List[str]) -> bool:
        if name not in self.sections:
            return False
        self.sections[name]['admin users'] = ' '.join(users)
        self._save()
        return True

    def set_invalid_users(self, name: str, users: List[str]) -> bool:
        if name not in self.sections:
            return False
        self.sections[name]['invalid users'] = ' '.join(users)
        self._save()
        return True

    def set_share_abse(self, name: str, enabled: bool) -> bool:
        """เปิด/ปิด ABSE per share"""
        if name not in self.sections:
            return False
        self.sections[name]['access based share enum'] = 'yes' if enabled else 'no'
        self._save()
        logger.info(f"Share '{name}' ABSE set to: {enabled}")
        return True
