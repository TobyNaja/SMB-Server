from fastapi import APIRouter, HTTPException
from pydantic import BaseModel, Field
from services.docker_executor import DockerExecutor
from config import settings
import logging

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/api/users", tags=["users"])

class UserCreate(BaseModel):
    username: str = Field(..., min_length=1, max_length=32)
    password: str = Field(..., min_length=1)
    fullname: str = ""

def executor():
    return DockerExecutor(settings.samba_container)

@router.get("")
async def list_users():
    try:
        users = executor().get_users()
        return {"users": users}
    except Exception as e:
        logger.error(f"list_users error: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.post("")
async def create_user(user: UserCreate):
    try:
        result = executor().create_user(user.username, user.password)
        if not result['success']:
            raise HTTPException(400, detail=result.get('error'))
        executor().reload_samba()
        return {"message": f"User {user.username} created"}
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(500, detail=str(e))

@router.delete("/{username}")
async def delete_user(username: str):
    try:
        executor().delete_user(username)
        executor().reload_samba()
        return {"message": f"User {username} deleted"}
    except Exception as e:
        raise HTTPException(500, detail=str(e))

@router.post("/{username}/password")
async def change_password(username: str, password: str):
    try:
        result = executor().set_password(username, password)
        if not result['success']:
            raise HTTPException(400, detail=result.get('error'))
        executor().reload_samba()
        return {"message": "Password updated"}
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(500, detail=str(e))
