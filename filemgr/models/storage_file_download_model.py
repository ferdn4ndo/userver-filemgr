import datetime
import math
import os
from django.db import models
from django.utils.translation import gettext_lazy as _
from rest_framework.request import Request

from filemgr.services.strings import generate_random_encoded_string
from .generic_model import GenericModel
from .user_model import CustomUser
from .storage_file_model import StorageFile


class StorageFileDownload(GenericModel):
    """
    Represents a download resource (a session of a download, having a user, a file and may expire)
    """
    storage_file = models.ForeignKey(
        to=StorageFile,
        on_delete=models.CASCADE,
        verbose_name=_("The file to be downloaded")
    )
    owner = models.ForeignKey(
        to=CustomUser,
        on_delete=models.SET_NULL,
        verbose_name=_("User that requested the download"),
        null=True
    )
    download_url = models.CharField(
        default=generate_random_encoded_string,
        max_length=255,
        verbose_name=_("Final url (should be signed) to the file to be downloaded")
    )
    created_at = models.DateTimeField(auto_now_add=True, verbose_name=_("Record creation timestamp"))

    @property
    def expires_at(self) -> datetime.datetime:
        """
        Gets a download expiration datetime for a given file based on a calculation on it's size.

        Adjust this value by altering the env DOWNLOAD_EXP_BYTES_SECS_RATIO.
        We recommend values between 4 and 5.

        The lowest size considered in the calculation is 100.000 bytes

        Some examples:

        Ratio   | Size                        | Time delta
        --------|-----------------------------|---------------------
        4       | 1872566538 bytes (1.74 GB)  | 7392 secs (2:03:12)
        4       | 108360999 bytes (103 MB)    | 4168 secs (1:09:28)
        4       | 156144 bytes (152 KB)       | 727 secs (0:12:07)
        --------|-----------------------------|---------------------
        4.25    | 1872566538 bytes (1.74 GB)  | 12899 secs (3:34:59)
        4.25    | 108360999 bytes (103 MB)    | 7017 secs (1:56:57)
        4.25    | 156144 bytes (152 KB)       | 1098 secs (0:18:18)
        --------|-----------------------------|---------------------
        4.5     | 1872566538 bytes (1.74 GB)  | 22509 secs (6:15:09)
        4.5     | 108360999 bytes (103 MB)    | 11814 secs (3:16:54)
        4.5     | 156144 bytes (152 KB)       | 1657 secs (0:27:37)
        --------|-----------------------------|---------------------
        4.75    | 1872566538 bytes (1.74 GB)  | 39280 secs (10:54:40)
        4.75    | 108360999 bytes (103 MB)    | 19891 secs (5:31:31)
        4.75    | 156144 bytes (152 KB)       | 2503 secs (0:41:43)
        --------|-----------------------------|---------------------
        5       | 1872566538 bytes (1.74 GB)  | 68544 secs (19:02:24)
        5       | 108360999 bytes (103 MB)    | 33488 secs (9:18:08)
        5       | 156144 bytes (152 KB)       | 3778 secs (0:12:07)
        --------|-----------------------------|---------------------

        :return:
        """
        lower_threshold = 100000
        bytes_to_secs_ratio = float(os.environ['DOWNLOAD_EXP_BYTES_SECS_RATIO'])
        file_size = lower_threshold if self.storage_file.size < lower_threshold else self.storage_file.size
        secs = math.pow(math.log(file_size, 10), bytes_to_secs_ratio)
        return self.created_at + datetime.timedelta(seconds=secs)


class DownloadHit(GenericModel):
    """
    Represents a download hit (every access to a download should create a record here)
    """
    download = models.ForeignKey(to=StorageFileDownload, on_delete=models.CASCADE, editable=False)
    user = models.ForeignKey(to=CustomUser, on_delete=models.SET_NULL, null=True)
    ip = models.GenericIPAddressField(null=True)
    user_agent = models.CharField(max_length=255, null=True)
    created_at = models.DateTimeField(auto_now_add=True, verbose_name=_("Record creation timestamp"))

    @staticmethod
    def create_from_download_request(download, request: Request):
        x_forwarded_for = request.META.get('HTTP_X_FORWARDED_FOR')
        ip = x_forwarded_for if x_forwarded_for else request.META.get('REMOTE_ADDR')
        user_agent = request.META['HTTP_USER_AGENT'] if 'HTTP_USER_AGENT' in request.META else 'Unknown'
        hit = DownloadHit(download=download, ip=ip, user_agent=user_agent)
        hit.save()
        return hit
