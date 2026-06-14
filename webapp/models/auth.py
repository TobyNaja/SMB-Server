from pydantic import BaseModel, Field
from typing import Optional
from datetime import datetime

class LoginRequest(BaseModel):
    """ฟอร์ม Login"""
    username: str = Field(..., min_length=1, max_length=32)
    password: str = Field(..., min_length=1)

class TokenResponse(BaseModel):
    """Response เมื่อ Login สำเร็จ"""
    access_token: str
    token_type: str = "bearer"
    expires_in: int

class UserInfo(BaseModel):
    """ข้อมูล User ที่ login"""
    username: str
    is_admin: bool = True
    expires_at: datetime

class TokenData(BaseModel):
    """ข้อมูล JWT Token"""
    username: str
    exp: datetime

class AdminCredential(BaseModel):
    """Admin account storage"""
    username: str
    hashed_password: str
    created_at: datetime
    last_login: Optional[datetime] = None
