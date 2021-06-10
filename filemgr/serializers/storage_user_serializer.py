from rest_framework import serializers

from filemgr.models import Storage, StorageUser, CustomUser


class StorageUserSerializer(serializers.ModelSerializer):
    storage = serializers.PrimaryKeyRelatedField(
        queryset=Storage.objects.all(),
        required=True,
        write_only=True
    )
    user = serializers.PrimaryKeyRelatedField(
        queryset=CustomUser.objects.all(),
        required=True,
    )
    created_by = serializers.SlugRelatedField(
        queryset=CustomUser.objects.all(),
        slug_field='username'
    )

    class Meta:
        model = StorageUser
        fields = [
            'id',
            'storage',
            'user',
            'may_write',
            'may_read',
            'created_at',
            'created_by',
        ]
