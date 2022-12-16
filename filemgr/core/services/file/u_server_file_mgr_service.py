import os

from api.exceptions.precondition_failed_exception import PreconditionFailedException
from core.services.web_request.web_request_service import WebRequestService


def get_file_mgr_url(endpoint='/') -> str:
    return 'http://' + os.environ['USERVER_FILEMGR_HOST'] + endpoint


def check_file_id_exists(file_id: str, auth_token: str) -> bool:
    file_url = get_file_mgr_url('/storages/{}/files/{}/'.format(os.environ['USERVER_FILEMGR_STORAGE_ID'], file_id))
    request = WebRequestService(
        url=file_url,
        method='GET',
        headers={
            'Authorization': 'Bearer {}'.format(auth_token)
        }
    )

    status_code = request.get_status_code()
    if status_code not in [200, 404]:
        raise PreconditionFailedException(
            'Unknown response from filemgr! Code: {} Data: {}'.format(status_code, str(request.get_json_response()))
        )

    return request.get_status_code() == 200
