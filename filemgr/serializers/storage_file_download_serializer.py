from django.utils.timezone import now

from rest_framework import serializers

from filemgr.drivers import load_storage_file_driver
from filemgr.models import StorageFileDownload, CustomUser, StorageFile


class StorageFileDownloadSerializer(serializers.ModelSerializer):
    storage_file = serializers.PrimaryKeyRelatedField(
        queryset=StorageFile.objects.all(),
        write_only=True,
        required=True
    )
    owner = serializers.PrimaryKeyRelatedField(
        queryset=CustomUser.objects.all(),
        write_only=True,
        required=True
    )
    created_at = serializers.ReadOnlyField()
    expires_at = serializers.ReadOnlyField()
    expired = serializers.SerializerMethodField()
    download_url = serializers.SerializerMethodField()

    class Meta:
        model = StorageFileDownload
        fields = [
            'id',
            'storage_file',
            'owner',
            'created_at',
            'expires_at',
            'expired',
            'download_url'
        ]

    def get_download_url(self, obj: StorageFileDownload) -> str:
        """
        Retrieves the download link for the file
        :param obj:
        :return:
        """
        driver = load_storage_file_driver(obj.storage_file)
        return driver.get_download_url(obj.storage_file, force_download=True)

    def get_expired(self, obj: StorageFileDownload) -> bool:
        return now() > obj.expires_at
