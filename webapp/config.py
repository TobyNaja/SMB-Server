from pydantic_settings import BaseSettings
import os
from datetime import timedelta

class Settings(BaseSettings):
    # API Server
    api_host: str = os.getenv("API_HOST", "0.0.0.0")
    api_port: int = int(os.getenv("API_PORT", "8080"))
    
    # Docker
    samba_container: str = os.getenv("SAMBA_CONTAINER", "samba-server")
    samba_config_path: str = os.getenv("SAMBA_CONFIG_PATH", "/etc/samba")
    samba_data_path: str = os.getenv("SAMBA_DATA_PATH", "/var/lib/samba")
    
    # Security & Authentication
    secret_key: str = os.getenv("SECRET_KEY", "dev-secret-key-change-in-production")
    algorithm: str = "HS256"
    access_token_expire_minutes: int = 1440  # 24 hours
    
    # Audit & Logging
    audit_log_path: str = os.getenv("AUDIT_LOG_PATH", "/mnt/shared/audit.json")
    admin_credentials_file: str = os.getenv("ADMIN_CREDS_FILE", "/mnt/shared/.admin")
    
    class Config:
        env_file = ".env"

settings = Settings()
