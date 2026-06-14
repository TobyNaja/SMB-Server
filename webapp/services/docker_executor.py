import docker
import shlex
import re
import logging

logger = logging.getLogger(__name__)

class DockerExecutor:
    def __init__(self, container_name: str):
        self.container_name = container_name
        try:
            self.client = docker.from_env()
            self.container = self.client.containers.get(container_name)
        except Exception as e:
            logger.error(f"Docker connection error: {e}")
            raise

    def execute(self, command: str) -> dict:
        try:
            result = self.container.exec_run(
                cmd=["/bin/bash", "-c", command],
                stdout=True,
                stderr=True,
                user="root"
            )
            output = result.output.decode("utf-8", errors="replace") if result.output else ""
            return {
                "success": result.exit_code == 0,
                "exit_code": result.exit_code,
                "output": output
            }
        except Exception as e:
            logger.error(f"exec error: {e}")
            return {"success": False, "exit_code": -1, "output": str(e)}

    def create_user(self, username: str, password: str) -> dict:
        if not re.match(r'^[a-zA-Z0-9_-]{1,32}$', username):
            return {"success": False, "error": "Invalid username format"}

        # สร้าง system user
        self.execute(
            f"id {shlex.quote(username)} > /dev/null 2>&1 || "
            f"useradd -m -s /usr/sbin/nologin {shlex.quote(username)}"
        )

        # ตั้ง samba password
        escaped = password.replace("'", "'\\''")
        result = self.execute(
            f"printf '{escaped}\\n{escaped}\\n' | "
            f"smbpasswd -a -s {shlex.quote(username)}"
        )
        if result['success']:
            return {"success": True, "message": f"User {username} created"}
        return {"success": False, "error": result['output']}

    def delete_user(self, username: str) -> dict:
        self.execute(f"smbpasswd -x {shlex.quote(username)} 2>/dev/null || true")
        self.execute(f"userdel -r {shlex.quote(username)} 2>/dev/null || true")
        return {"success": True, "message": f"User {username} deleted"}

    def set_password(self, username: str, password: str) -> dict:
        escaped = password.replace("'", "'\\''")
        result = self.execute(
            f"printf '{escaped}\\n{escaped}\\n' | "
            f"smbpasswd -s {shlex.quote(username)}"
        )
        if result['success']:
            return {"success": True, "message": "Password updated"}
        return {"success": False, "error": result['output']}

    def create_group(self, group_name: str) -> dict:
        self.execute(f"groupadd {shlex.quote(group_name)} 2>/dev/null || true")
        return {"success": True, "message": f"Group {group_name} created"}

    def add_user_to_group(self, username: str, group_name: str) -> dict:
        result = self.execute(
            f"usermod -a -G {shlex.quote(group_name)} {shlex.quote(username)}"
        )
        if result['success']:
            return {"success": True, "message": f"Added {username} to {group_name}"}
        return {"success": False, "error": result['output']}

    def remove_user_from_group(self, username: str, group_name: str) -> dict:
        result = self.execute(
            f"gpasswd -d {shlex.quote(username)} {shlex.quote(group_name)}"
        )
        if result['success']:
            return {"success": True, "message": f"Removed {username} from {group_name}"}
        return {"success": False, "error": result['output']}

    def get_users(self) -> list:
        result = self.execute("pdbedit -L 2>/dev/null")
        users = []
        if result['success']:
            for line in result['output'].split('\n'):
                line = line.strip()
                if not line:
                    continue
                parts = line.split(':')
                if len(parts) >= 2:
                    users.append({
                        "username": parts[0],
                        "uid": parts[1],
                        "fullname": parts[2].strip() if len(parts) > 2 else "",
                        "disabled": False
                    })
        return users

    def get_groups(self) -> list:
        result = self.execute("getent group")
        groups = []
        skip = {
            'root','bin','sys','daemon','adm','lp','mail','news',
            'uucp','man','proxy','kmem','dialout','fax','voice',
            'cdrom','floppy','tape','sudo','audio','dip','www-data',
            'backup','operator','list','irc','src','gnats','shadow',
            'utmp','video','sasl','plugdev','staff','games','users',
            'nogroup','crontab','syslog','tty','disk','input','netdev',
            'render','sgx','kvm','messagebus','samba','sambashare'
        }
        if result['success']:
            for line in result['output'].split('\n'):
                if not line.strip():
                    continue
                name = line.split(':')[0]
                if name not in skip and not name.startswith('_'):
                    groups.append(name)
        return sorted(groups)

    def reload_samba(self) -> dict:
        self.execute("smbcontrol all reload-config 2>/dev/null || pkill -HUP smbd || true")
        return {"success": True, "message": "Samba reloaded"}
