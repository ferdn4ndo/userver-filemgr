from rest_framework import serializers

from filemgr.models import Storage
from filemgr.models.storage_file_model import StorageFile, StorageFileMimeType
from filemgr.serializers.user_serializer import UserSerializer


class StorageFileMimeTypeSerializer(serializers.ModelSerializer):

    class Meta:
        model = StorageFileMimeType
        fields = [
            'mime_type',
            'generic_type',
            'description',
            'extensions',
        ]


class StorageFileSerializer(serializers.ModelSerializer):
    signature_key = serializers.ReadOnlyField()
    storage = serializers.PrimaryKeyRelatedField(
        write_only=True,
        required=True,
        queryset=Storage.objects.all
    )
    owner = UserSerializer()
    status = serializers.ReadOnlyField()
    size = serializers.ReadOnlyField()
    hash = serializers.ReadOnlyField()
    type = StorageFileMimeTypeSerializer(read_only=True)
    origin = serializers.ReadOnlyField()
    original_path = serializers.ReadOnlyField()
    available = serializers.ReadOnlyField()
    excluded = serializers.ReadOnlyField()

    class Meta:
        model = StorageFile
        fields = [
            'id',
            'signature_key',
            'storage',
            'owner',
            'status',
            'visibility',
            'size',
            'hash',
            'type',
            'metadata',
            'origin',
            'original_path',
            'real_filepath',
            'virtual_filepath',
            'available',
            'excluded',
            'created_by',
            'created_at',
            'updated_by',
            'updated_at'
        ]
