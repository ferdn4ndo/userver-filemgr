import magic
import mimetypes
import os
from abc import ABC, abstractmethod
from pathlib import Path
from typing import Union
from urllib.parse import urlparse

from django.core.files.uploadedfile import TemporaryUploadedFile, InMemoryUploadedFile
from django.utils.translation import gettext_lazy as _

from filemgr.errors import UnexpectedCondition
from filemgr.models.storage_file_model import StorageFile, StorageFileMimeType
from filemgr.models.storage_model import Storage
from filemgr.models.user_model import CustomUser
from filemgr.services.file import generate_file_hash, save_from_memory
from filemgr.services.web_request import WebRequest


class GenericDriver(ABC):
    """
    Abstract class representing a generic storage driver
    """

    def __init__(self, storage: Storage):
        """
        Class constructor
        """
        self.storage = storage

    @abstractmethod
    def get_remote_root_path(self) -> str:
        """
        Returns the remote root path. Should be implemented for each storage.
        :return: The root path
        """
        raise NotImplementedError("To be implemented.")

    @abstractmethod
    def perform_upload_from_path(self, local_path: str, remote_path: str):
        """
        Upload a file from a local path
        :param local_path: the local path of the file that's going to be uploaded
        :param remote_path: the real remote path to write the file
        """
        raise NotImplementedError("To be implemented.")

    @abstractmethod
    def perform_upload_from_url(self, url: str, remote_path: str):
        """
        Upload a file from a given url
        :param url: the url of the file that's going to be uploaded
        :param remote_path: the real remote path to write the file
        :return:
        """
        raise NotImplementedError("To be implemented.")

    @abstractmethod
    def perform_download_to_path(self, remote_path: str, local_dest: str):
        """
        Upload a file from a given url (should be implemented)
        :param remote_path: the real remote path from where the file will be downloaded
        :param local_dest: the destination file path of the download
        :return:
        """
        raise NotImplementedError("To be implemented.")

    @abstractmethod
    def get_download_url(self, file: StorageFile, expiration_seconds: int = 3600, force_download: bool = False):
        """
        Retrieve an expirable download url for a given storage file
        :param file: the StorageFile model to be downloaded
        :param force_download: if the file download should be enforced (not rendered)
        :param expiration_seconds: time in seconds for the URL to remain valid. 0 for the infinite and beyond.
        :return:
        """
        raise NotImplementedError("To be implemented.")

    @abstractmethod
    def perform_delete(self, file: StorageFile):
        """
        Performs the remote file deletion given a StorageFile. Implement this method with the storage logic.
        :param file: the StorageFile model to be deleted
        :return:
        """
        raise NotImplementedError("To be implemented.")

    @abstractmethod
    def remote_file_exists(self, file: StorageFile) -> bool:
        """
        Determines if a given real remote path exists
        :param file: the file to check whether its remote path exists
        :return: boolean indicating if the file exists or not
        """
        raise NotImplementedError("To be implemented.")

    def get_real_remote_path(self, file: StorageFile) -> str:
        """
        Get the real remote path for a StoreFile (usually different from the 'visual' remote_path)
        :param file: the file to retrieve the path
        :return: the file path
        """
        if self.get_remote_root_path() != "":
            return "{}/{}".format(self.get_remote_root_path(), str(file.id))

        return str(file.id)

    def upload_from_path(
            self,
            user: CustomUser,
            path: str,
            virtual_path: str = "",
            original_path: str = "",
            overwrite: bool = False,
            visibility: str = StorageFile.FileVisibility.USER,
            is_url: bool = False,
    ) -> StorageFile:
        """
        Upload a file from a local path or an URL. Please do not modify this method. Instead, put the storage-specific
        upload instructions in the "perform_upload_from_path" or "perform_upload_from_url" method, respectively.
        :param user: the user performing the upload (owner of the file)
        :param path: the file path (local or url) that's going to be uploaded
        :param virtual_path: the virtual remote path to where the file should be uploaded
        :param overwrite: if the file should be replaced in case the remote path already exists
        :param visibility: file visibility (one of StorageFile.FileVisibility options)
        :param is_url: if the "path" argument is an URL (True) or a local file path (False)
        :return:
        """
        file = self.create_empty_file_resource(user)
        file.visibility = visibility
        file.original_path = original_path
        file.owner = user
        file.virtual_filepath = self.generate_virtual_path(file, virtual_path=virtual_path, is_url=is_url)
        file.real_filepath = self.generate_real_path(file)

        # Check if the virtual_path (where the file should be uploaded) doesn't already exists
        self.check_if_virtual_path_exists(file.virtual_filepath, raise_ex_if_yes=not overwrite)

        # Call the appropriate file metadata reader to update the model
        if is_url:
            self.update_file_info_from_url(file=file, file_url=path)
        else:
            self.update_file_info_from_local(file=file, local_path=path)
        file.save()

        self.check_if_virtual_path_exists(file.virtual_filepath, raise_ex_if_no=True)

        # Perform the upload by calling the appropriate method
        file_real_path = self.get_real_remote_path(file)
        if is_url:
            self.perform_upload_from_url(url=path, remote_path=file_real_path)
        else:
            self.perform_upload_from_path(local_path=path, remote_path=file_real_path)

        # Check if the file was really uploaded
        self.check_if_virtual_path_exists(file.virtual_filepath, raise_ex_if_no=True)

        file.status = StorageFile.FileStatus.QUEUED
        file.save()
        return file

    def upload_from_request_file(
            self,
            user: CustomUser,
            request_file: Union[TemporaryUploadedFile, InMemoryUploadedFile],
            virtual_path: str = "",
            overwrite: bool = False,
            visibility: str = StorageFile.FileVisibility.USER,
            is_url: bool = False,
    ) -> StorageFile:
        """
        Upload a file from an upload request file object. In case it's in memory (for small files), persists the file
        on disk for further uploading through a normal file
        :param user: the user performing the upload (owner of the file)
        :param request_file: the request file object returned from the upload request
        :param virtual_path: the virtual remote path to where the file should be uploaded
        :param overwrite: if the file should be replaced in case the remote path already exists
        :param visibility: file visibility (one of StorageFile.FileVisibility options)
        :param is_url: if the "path" argument is an URL (True) or a local file path (False)
        :return:
        """
        if isinstance(request_file, TemporaryUploadedFile):
            temp_file_path = request_file.temporary_file_path()
        elif isinstance(request_file, InMemoryUploadedFile):
            temp_file_path = save_from_memory(request_file)
        else:
            raise UnexpectedCondition(
                _("The request_file should be either a TemporaryUploadedFile or a InMemoryUploadedFile")
            )
        file = self.upload_from_path(
            user=user,
            path=temp_file_path,
            virtual_path=virtual_path,
            overwrite=overwrite,
            visibility=visibility,
            is_url=is_url
        )

        file.original_path = request_file.name
        file.save()

        return file

    def download_to_path(self, file: StorageFile, local_path: str, replace: bool = False):
        """
        Performs the download of a StorageFile to a local path
        Upload a file from a given url (should be implemented)
        :param file: the file (StorageFile model) to be downloaded
        :param local_path: the destination file path of the download
        :param replace: if the file should be replaced in case the local path already exists
        :return:
        """
        # Check if the local_path (where the file should be downloaded) doesn't already exists
        self.check_if_local_file_exists(local_path, raise_ex_if_yes=not replace)

        self.perform_download_to_path(self.get_real_remote_path(file), local_path)

        # Check if the file was really downloaded
        self.check_if_local_file_exists(local_path, raise_ex_if_no=True)

    def delete_file(self, file: StorageFile) -> bool:
        """
        Remotely deletes a StorageFile. Please do not modify this method. Instead, put the
        storage-specific delete instructions in the "perform_delete" method.
        :param file: the StorageFile model to be deleted
        :return:
        """
        if not self.remote_file_exists(file):
            return False

        self.perform_delete(file)
        file_exists = self.remote_file_exists(file)
        if file_exists:
            raise UnexpectedCondition("File wasn't deleted remotely")

        file.status = StorageFile.FileStatus.DELETED
        file.available = False
        file.excluded = True
        file.save()
        return True

    def virtual_path_exists(self, virtual_path) -> bool:
        """
        Determines if a given virtual path exists
        :param virtual_path:
        :return: boolean indicating if the file exists or not
        """
        file_count = StorageFile.objects.filter(
            storage=self.storage,
            virtual_filepath=virtual_path,
            excluded=False,
            available=True
        ).count()
        return True if file_count > 0 else False

    def check_if_virtual_path_exists(
            self,
            virtual_path: str,
            raise_ex_if_no: bool = False,
            raise_ex_if_yes: bool = False,
    ) -> bool:
        """
        Checks if a given virtual_path exists remotely, returning a boolean accordingly.
        However, if raise_ex_if_no is True and the file doesn't exist, a FileNotFoundError will be raised.
        Similarly, if raise_ex_if_yes is True and the file does exist, a FileExistsError will be raised.

        If the file doesn't exist and exist at the same time (are we quantum already?), the FileNotFoundError will
        be raised first.

        :param virtual_path: the path do be checked
        :param raise_ex_if_no: will raise a FileNotFoundError if the virtual_path doesn't exist
        :param raise_ex_if_yes: will raise a FileExistsError if the virtual_path does exist
        :return:
        """
        virtual_path_exists_bool = self.virtual_path_exists(virtual_path)

        if not virtual_path_exists_bool and raise_ex_if_no:
            raise FileNotFoundError(
                _("The file path '{}' should remotely exists and wasn't found.").format(virtual_path)
            )
        if virtual_path_exists_bool and raise_ex_if_yes:
            raise FileExistsError(
                _("The file path '{}' should not remotely exists but it does.").format(virtual_path)
            )
        return virtual_path_exists_bool

    @staticmethod
    def check_if_local_file_exists(
            local_path: str,
            raise_ex_if_no: bool = False,
            raise_ex_if_yes: bool = False,
    ) -> bool:
        """
        Checks if a given local_path exists locally, returning a boolean accordingly.
        However, if raise_ex_if_no is True and the file doesn't exist, a FileNotFoundError will be raised.
        Similarly, if raise_ex_if_yes is True and the file does exist, a FileExistsError will be raised.

        If the file doesn't exist and exist at the same time (are we quantum already?), the FileNotFoundError will
        be raised first.

        :param local_path: the path do be checked
        :param raise_ex_if_no: will raise a FileNotFoundError if the virtual_path doesn't exist
        :param raise_ex_if_yes: will raise a FileExistsError if the virtual_path does exist
        :return:
        """
        local_path_exists_bool = os.path.isfile(local_path)

        if not local_path_exists_bool and raise_ex_if_no:
            raise FileNotFoundError(
                _("The file path '{}' should locally exists and wasn't found.").format(local_path)
            )
        if local_path_exists_bool and raise_ex_if_yes:
            raise FileExistsError(
                _("The file path '{}' should locally exists and wasn't found.").format(local_path)
            )
        return local_path_exists_bool

    def create_empty_file_resource(self, user: CustomUser) -> StorageFile:
        """
        Create an empty file resource (database only)
        :param user:
        :return:
        """
        file = StorageFile(
            storage=self.storage,
            created_by=user,
            updated_by=user,
        )
        file.save()

        if self.remote_file_exists(file):
            raise FileExistsError(_("Remote real file '{}' already exists (and should not)!").format(
                self.get_real_remote_path(file)
            ))

        return file

    def generate_virtual_path(self, file: StorageFile, virtual_path: str = None, is_url: bool = False):
        """
        Generate a remote virtual path for a given file
        :param file: The file resource being populated
        :param virtual_path: The filepath/url to base the filename (if auto)
        :param is_url: if its an url (true) or a filepath (false)
        :return:
        """
        if virtual_path is None or virtual_path == "":
            return file.id

        if is_url:
            url_parser = urlparse(virtual_path)
            return os.path.basename(url_parser.path)

        return virtual_path

    def generate_real_path(self, file: StorageFile) -> str:
        """
        Generate a remote real path for a given file
        :param file: The file resource being populated
        :return:
        """
        return os.path.join(self.get_remote_root_path(), str(file.id))

    @staticmethod
    def update_file_info_from_local(file: StorageFile, local_path: str):
        """
        Update a StorageFile resource metadata based on a local file path
        :param file: The StorageFile resource going to be updated
        :param local_path: The file path to read information from
        """
        file.size = int(Path(local_path).stat().st_size)
        mime = magic.Magic(mime=True)
        guessed_mime = mime.from_file(local_path)
        if guessed_mime is not None:
            file.type = StorageFileMimeType.from_mime_type(guessed_mime)
        file.origin = StorageFile.FileOrigin.LOCAL
        file.original_filename = os.path.split(local_path)[1]
        file.original_path = local_path
        file.hash = generate_file_hash(local_path)
        file.save()

    @staticmethod
    def update_file_info_from_url(file: StorageFile, file_url: str):
        """
        Update a StorageFile resource metadata based on a local file path
        :param file: The StorageFile resource going to be updated
        :param file_url: The file url to read information from
        """
        url_info = WebRequest.get_url_info(url=file_url)
        url_address_parser = urlparse(file_url)

        file.size = int(url_info['Content-Length'])
        file.type = StorageFileMimeType.from_mime_type(url_info['Content-Type'])
        file.origin = StorageFile.FileOrigin.WEB
        file.original_filename = os.path.basename(url_address_parser.path)
        file.original_path = file_url
        file.save()
