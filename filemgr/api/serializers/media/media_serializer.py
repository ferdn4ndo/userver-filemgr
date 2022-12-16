from api.exceptions.internal_server_exception import InternalServerException
from core.models.media.media_document_model import MediaDocument
from core.models.media.media_image_model import MediaImage
from core.models.media.media_model import Media
from core.models.media.media_video_model import MediaVideo

from api.serializers.generic_audited_model_serializer import GenericAuditedModelSerializer
from api.serializers.media.media_document_serializer import MediaDocumentSerializer
from api.serializers.media.media_image_serializer import MediaImageSerializer
from api.serializers.media.media_video_serializer import MediaVideoSerializer
from api.serializers.user.user_serializer import UserSerializer


class MediaSerializer(GenericAuditedModelSerializer):
    author = UserSerializer()
    created_by = UserSerializer()

    class Meta:
        model = Media
        fields = [
            'id',
            'title',
            'type',
            'description',
            'storage_file',
            'created_at',
            'created_by',
            'updated_at',
            'updated_by',
        ]

    def to_representation(self, instance: Media):
        instance_dict = super(MediaSerializer, self).to_representation(instance)
        instance_dict['metadata'] = {}

        if instance.type == Media.MediaType.IMAGE:
            model = MediaImage
            serializer = MediaImageSerializer
        elif instance.type == Media.MediaType.VIDEO:
            model = MediaVideo
            serializer = MediaVideoSerializer
        else:
            return instance_dict

        image_queryset = model.objects.filter(media_item=instance)
        if image_queryset.count() == 1:
            instance_dict['metadata'] = serializer(image_queryset[0]).data
        else:
            raise InternalServerException(
                "Database inconsistency: found {} {} objects for Media ID {} (must have exactly one)".format(
                    image_queryset.count(), model.__class__, instance.id,
                )
            )

        return instance_dict
