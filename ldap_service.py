import subprocess
import re
import base64
import logging
from typing import List, Dict, Optional
from config import settings

logger = logging.getLogger(__name__)

class LdapService:

    @staticmethod
    def _ldapsearch(base: str, scope: str, filter_str: str, attrs: List[str]) -> str:
        from services.docker_executor import DockerExecutor
        ex = DockerExecutor(settings.samba_container)
        attrs_str = " ".join(attrs) if attrs else ""
        cmd = (
            f"ldapsearch "
            f"-H ldap://{settings.ldap_server}:{settings.ldap_port} "
            f"-D '{settings.ldap_bind_dn}' "
            f"-w '{settings.ldap_bind_pw}' "
            f"-b '{base}' "
            f"-s {scope} "
            f"'{filter_str}' "
            f"{attrs_str} 2>&1"
        )
        result = ex.execute(cmd)
        return result.get('output', '')

    @staticmethod
    def _decode_value(value: str) -> str:
        """Decode base64 value ถ้าเริ่มด้วย ::"""
        try:
            return base64.b64decode(value).decode('utf-8', errors='replace').strip()
        except Exception:
            return value

    @staticmethod
    def _parse_ldap_entries(output: str) -> List[Dict]:
        """Parse ldapsearch output รองรับ base64 และ multi-line"""
        entries = []
        current = {}
        last_key = None

        for line in output.split('\n'):
            # continuation line (space indent = multi-line value)
            if line.startswith(' ') and last_key and current:
                current[last_key] = current[last_key] + line.strip()
                continue

            line = line.strip()

            # blank line = end of entry
            if not line:
                if current and 'dn' in current:
                    entries.append(current)
                    current = {}
                    last_key = None
                continue

            # skip comments
            if line.startswith('#'):
                continue

            if ':' in line:
                # base64 encoded: key:: value
                if ':: ' in line:
                    key, _, value = line.partition(':: ')
                    key = key.strip().lower()
                    decoded = LdapService._decode_value(value.strip())
                    last_key = key
                    if key in current:
                        if isinstance(current[key], list):
                            current[key].append(decoded)
                        else:
                            current[key] = [current[key], decoded]
                    else:
                        current[key] = decoded
                else:
                    key, _, value = line.partition(': ')
                    key = key.strip().lower()
                    value = value.strip()
                    last_key = key
                    if key in current:
                        if isinstance(current[key], list):
                            current[key].append(value)
                        else:
                            current[key] = [current[key], value]
                    else:
                        current[key] = value

        if current and 'dn' in current:
            entries.append(current)

        return entries

    @staticmethod
    def search_users(query: str = "", ou: str = None, limit: int = 50) -> List[Dict]:
        try:
            base = f"{ou},{settings.ldap_base_dn}" if ou else settings.ldap_base_dn
            if query:
                filter_str = (
                    f"(&(objectClass=user)(objectCategory=person)"
                    f"(!(userAccountControl:1.2.840.113556.1.4.803:=2))"
                    f"(|(sAMAccountName=*{query}*)(cn=*{query}*)(mail=*{query}*)))"
                )
            else:
                filter_str = (
                    "(&(objectClass=user)(objectCategory=person)"
                    "(!(userAccountControl:1.2.840.113556.1.4.803:=2)))"
                )

            output = LdapService._ldapsearch(
                base=base, scope="sub", filter_str=filter_str,
                attrs=["sAMAccountName", "cn", "mail", "department", "title", "distinguishedName"]
            )

            entries = LdapService._parse_ldap_entries(output)
            users = []

            for e in entries:
                sam = e.get('samaccountname', '')
                if not sam or sam.endswith('$'):
                    continue

                dn = e.get('dn', '')
                # ถ้า dn เป็น base64 ให้ decode
                if dn.startswith(':'):
                    dn = LdapService._decode_value(dn.lstrip(': '))

                ou_match = re.search(r'OU=([^,]+)', dn, re.IGNORECASE)
                user_ou = ou_match.group(1) if ou_match else 'Users'

                cn_val = e.get('cn', sam)
                if isinstance(cn_val, list):
                    cn_val = cn_val[0]

                mail_val = e.get('mail', '')
                if isinstance(mail_val, list):
                    mail_val = mail_val[0]

                users.append({
                    "username": sam,
                    "display_name": cn_val,
                    "email": mail_val,
                    "department": e.get('department', ''),
                    "title": e.get('title', ''),
                    "ou": user_ou,
                    "source": "ad"
                })

                if len(users) >= limit:
                    break

            return users

        except Exception as e:
            logger.error(f"LDAP search_users error: {e}")
            return []

    @staticmethod
    def search_users_by_ou(ou_name: str, limit: int = 100) -> List[Dict]:
        """ดู users ใน OU เฉพาะ"""
        ou = f"OU={ou_name}"
        return LdapService.search_users(ou=ou, limit=limit)

    @staticmethod
    def search_groups(query: str = "", limit: int = 50) -> List[Dict]:
        try:
            filter_str = f"(&(objectClass=group)(cn=*{query}*))" if query else "(objectClass=group)"
            output = LdapService._ldapsearch(
                base=settings.ldap_base_dn, scope="sub",
                filter_str=filter_str,
                attrs=["cn", "description", "distinguishedName"]
            )

            entries = LdapService._parse_ldap_entries(output)
            groups = []

            for e in entries:
                cn = e.get('cn', '')
                if not cn:
                    continue

                dn = e.get('dn', '')
                ou_match = re.search(r'OU=([^,]+)', dn, re.IGNORECASE)
                group_ou = ou_match.group(1) if ou_match else 'Groups'

                desc = e.get('description', '')
                if isinstance(desc, list):
                    desc = desc[0]

                groups.append({
                    "name": cn,
                    "description": desc[:80] if desc else '',
                    "ou": group_ou,
                    "smb_name": f"@{cn}",
                    "source": "ad"
                })

                if len(groups) >= limit:
                    break

            return groups

        except Exception as e:
            logger.error(f"LDAP search_groups error: {e}")
            return []

    @staticmethod
    def get_user(username: str) -> Optional[Dict]:
        try:
            output = LdapService._ldapsearch(
                base=settings.ldap_base_dn, scope="sub",
                filter_str=f"(&(objectClass=user)(sAMAccountName={username}))",
                attrs=["sAMAccountName", "cn", "mail", "department", "title", "memberOf"]
            )
            entries = LdapService._parse_ldap_entries(output)
            if not entries:
                return None

            e = entries[0]
            member_of = e.get('memberof', [])
            if isinstance(member_of, str):
                member_of = [member_of]

            groups = []
            for g in member_of:
                cn_match = re.search(r'CN=([^,]+)', g)
                if cn_match:
                    groups.append(cn_match.group(1))

            return {
                "username": e.get('samaccountname', username),
                "display_name": e.get('cn', username),
                "email": e.get('mail', ''),
                "department": e.get('department', ''),
                "title": e.get('title', ''),
                "groups": groups,
                "source": "ad"
            }
        except Exception as e:
            logger.error(f"LDAP get_user error: {e}")
            return None

    @staticmethod
    def test_connection() -> Dict:
        try:
            output = LdapService._ldapsearch(
                base=settings.ldap_base_dn, scope="base",
                filter_str="(objectClass=*)", attrs=["dn"]
            )
            if "Invalid credentials" in output:
                return {"ok": False, "error": "Invalid credentials"}
            if "Can't contact" in output:
                return {"ok": False, "error": "Cannot connect"}
            if "dn:" in output.lower():
                return {"ok": True, "server": settings.ldap_server,
                        "base_dn": settings.ldap_base_dn, "domain": settings.ldap_domain}
            return {"ok": False, "error": output[:200]}
        except Exception as e:
            return {"ok": False, "error": str(e)}
