from rest_framework import serializers

from core.models.user.user_model import User


class UserOutputSerializer(serializers.ModelSerializer):
    class Meta:
        model = User
        fields = [
            'uuid',
            'username',
            'registered_at',
            'last_activity_at',
        ]
