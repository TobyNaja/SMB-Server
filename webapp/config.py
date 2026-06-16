from pydantic_settings import BaseSettings
import os

class Settings(BaseSettings):
    api_host: str = os.getenv("API_HOST", "0.0.0.0")
    api_port: int = int(os.getenv("API_PORT", "8080"))
    samba_container: str = os.getenv("SAMBA_CONTAINER", "samba-server")
    samba_config_path: str = os.getenv("SAMBA_CONFIG_PATH", "/etc/samba")
    samba_data_path: str = os.getenv("SAMBA_DATA_PATH", "/var/lib/samba")
    secret_key: str = os.getenv("SECRET_KEY", "dev-secret-key-change-in-production")
    algorithm: str = "HS256"
    access_token_expire_minutes: int = 1440
    audit_log_path: str = os.getenv("AUDIT_LOG_PATH", "/mnt/shared/audit.json")
    admin_credentials_file: str = os.getenv("ADMIN_CREDS_FILE", "/mnt/shared/.admin")

    # LDAP Settings
    ldap_server: str = os.getenv("LDAP_SERVER", "10.70.37.143")
    ldap_port: int = int(os.getenv("LDAP_PORT", "389"))
    ldap_base_dn: str = os.getenv("LDAP_BASE_DN", "DC=it,DC=kmitl,DC=ac,DC=th")
    ldap_bind_dn: str = os.getenv("LDAP_BIND_DN", "ldap-bind-nas@IT.KMITL.AC.TH")
    ldap_bind_pw: str = os.getenv("LDAP_BIND_PW", "Mephrfc-vl9tcdp-ruyhfth")
    ldap_domain: str = os.getenv("LDAP_DOMAIN", "IT.KMITL.AC.TH")

    class Config:
        env_file = ".env"

settings = Settings()
