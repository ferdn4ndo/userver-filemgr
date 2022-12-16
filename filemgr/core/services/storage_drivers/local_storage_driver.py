import os
import shutil
from shutil import copyfile

from core.services.storage_drivers.generic_storage_driver import GenericStorageDriver
from core.models.storage.storage_file_model import StorageFile
from core.services.web_request.web_request_service import WebRequestService


class LocalStorageDriver(GenericStorageDriver):
    """
    Storage driver class for local files
    """

    def get_remote_root_path(self) -> str:
        """
        Returns the remote root path. Should be implemented for each storage.
        :return: The root path
        """
        return os.environ['LOCAL_TEST_STORAGE_ROOT']

    def perform_upload_from_path(self, local_path: str, remote_path: str):
        """
        Upload a file from a local path
        :param local_path: the local path of the file that's going to be uploaded
        :param remote_path: the remote path to where the file should be uploaded
        """
        copyfile(local_path, remote_path)

    def perform_upload_from_url(self, url: str, remote_path: str):
        """
        Upload a file from a given url
        :param url: the url of the file that's going to be uploaded
        :param remote_path: the remote path to where the file should be uploaded
        """
        req_for_file = WebRequestService(url=url, stream=True)
        with req_for_file.object as r:
            r.raise_for_status()
            with open(remote_path, 'wb') as f:
                shutil.copyfileobj(r.raw, f)

    def perform_download_to_path(self, remote_path: str, local_dest: str):
        """
        Download a file from the remote storage
        :param remote_path: the real remote path from where the file will be downloaded
        :param local_dest: the destination file path of the download
        :return:
        """
        copyfile(remote_path, local_dest)

    def get_download_url(self, file: StorageFile, expiration_seconds: int = 3600, force_download: bool = False):
        """
        Retrieve an expirable download url for a given storage file
        :param file: the StorageFile model to be downloaded
        :param force_download: if the file download should be enforced (not rendered)
        :param expiration_seconds: time in seconds for the URL to remain valid. 0 for the infinite and beyond.
        :return:
        """
        return self.get_real_remote_path(file)

    def perform_delete(self, file: StorageFile):
        """
        Performs the remote file deletion given a StorageFile. Implement this method with the storage logic.
        :param file: the StorageFile model to be deleted
        :return:
        """
        os.remove(self.get_real_remote_path(file))

    def remote_file_exists(self, file: StorageFile) -> bool:
        """
        Determines if a given real remote path exists
        :param file: the file to check whether its remote path exists
        :return: boolean indicating if the file exists or not
        """
        return os.path.isfile(self.get_real_remote_path(file))
