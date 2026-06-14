from fastapi import APIRouter, HTTPException
from pydantic import BaseModel
from services.docker_executor import DockerExecutor
from config import settings

router = APIRouter(prefix="/api/groups", tags=["groups"])

class GroupCreate(BaseModel):
    group_name: str

def executor():
    return DockerExecutor(settings.samba_container)

@router.get("")
async def list_groups():
    return {"groups": executor().get_groups()}

@router.post("")
async def create_group(group: GroupCreate):
    result = executor().create_group(group.group_name)
    if not result['success']:
        raise HTTPException(400, detail=result.get('error'))
    return {"message": f"Group {group.group_name} created"}

@router.post("/{group_name}/members/{username}")
async def add_member(group_name: str, username: str):
    result = executor().add_user_to_group(username, group_name)
    if not result['success']:
        raise HTTPException(400, detail=result.get('error'))
    return {"message": f"Added {username} to {group_name}"}

@router.delete("/{group_name}/members/{username}")
async def remove_member(group_name: str, username: str):
    result = executor().remove_user_from_group(username, group_name)
    if not result['success']:
        raise HTTPException(400, detail=result.get('error'))
    return {"message": f"Removed {username} from {group_name}"}
