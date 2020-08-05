
from rest_framework import serializers

from filemgr.models.storage_file_download_model import StorageFileDownload
from filemgr.models.storage_file_model import StorageFile
from filemgr.models.user_model import CustomUser


class UserSerializer(serializers.ModelSerializer):
    total_downloads = serializers.SerializerMethodField(method_name='get_total_downloads')
    total_uploads = serializers.SerializerMethodField(method_name='get_total_uploads')

    class Meta:
        model = CustomUser
        fields = [
            'id',
            'username',
            'system_name',
            'is_admin',
            'is_active',
            'registered_at',
            'last_activity_at',
            'total_downloads',
            'total_uploads',
        ]

    def get_total_downloads(self, obj: CustomUser) -> int:
        """
        Returns the total amount of downloads recorded for a given user
        :param obj:
        :return: int with the total amount
        """
        return len(StorageFileDownload.objects.filter(owner=obj))

    def get_total_uploads(self, obj: CustomUser) -> int:
        """
        Returns the total amount of uploads recorded for a given user
        :param obj:
        :return: int with the total amount
        """
        return len(StorageFile.objects.filter(created_by=obj))


class UserInputSerializer(serializers.Serializer):
    username = serializers.CharField(max_length=255, required=True)
    password = serializers.CharField(max_length=255, required=True)


class UserOutputSerializer(serializers.ModelSerializer):
    class Meta:
        model = CustomUser
        fields = [
            'uuid',
            'username',
            'registered_at',
            'last_activity_at',
        ]
