from core.models.media.media_video_model import MediaVideo

from api.serializers.generic_audited_model_serializer import GenericAuditedModelSerializer


class MediaVideoSerializer(GenericAuditedModelSerializer):

    class Meta:
        model = MediaVideo
        fields = [
            'id',
            'fps',
            'duration',
            'size_tag',
            'height',
            'width',
            'created_at',
            'updated_at',
        ]
