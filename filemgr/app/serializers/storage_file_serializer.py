from rest_framework import serializers

from app.models import Storage
from app.models.storage_file_model import StorageFile, StorageFileMimeType
from app.serializers.user_serializer import UserSerializer


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
    real_path = serializers.ReadOnlyField()
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
            'real_path',
            'virtual_path',
            'available',
            'excluded',
            'created_by',
            'created_at',
            'updated_by',
            'updated_at'
        ]
