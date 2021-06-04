from rest_framework import serializers

from filemgr.models.storage_file_model import StorageFile


class StorageFileUploadSerializer(serializers.Serializer):
    virtual_path = serializers.CharField(max_length=1024, default="")
    overwrite = serializers.BooleanField(default=False)
    visibility = serializers.ChoiceField(
        choices=StorageFile.FileVisibility.choices,
        default=StorageFile.FileVisibility.USER
    )

    def update(self, instance, validated_data):
        pass

    def create(self, validated_data):
        pass


class StorageFileUploadUrlSerializer(StorageFileUploadSerializer):
    url = serializers.URLField()
