import json
import os
from typing import Any, Dict

from urllib.request import Request

from django.db import models
from rest_framework import status, serializers
from rest_framework.response import Response
from rest_framework.test import APITestCase, APIRequestFactory

from api.services.pagination import CustomPagination
from api.tests import dataset
from api.views.generic_model_view import GenericModelViewSet
from core.models import CustomUser


class GenericTestCase:

    class GenericTest(APITestCase):
        view: GenericModelViewSet or None = None
        factory: APIRequestFactory or None = APIRequestFactory()
        endpoint: str or None = None
        serializer: serializers.BaseSerializer or None
        model: models.Model or None
        requires_admin: bool = False
        user: CustomUser or None

        def setUp(self) -> None:
            self.user = dataset.create_user(is_admin=self.requires_admin)

        def tearDown(self) -> None:
            if self.user is not None:
                self.user.delete()

        def check_pre_conditions(self):
            if not hasattr(self, "model") or self.model is None:
                self.fail("The parameter 'model' must be set (and not None)")

            if not hasattr(self, "view") or self.view is None:
                self.fail("The parameter 'view' must be set (and not None)")

        @staticmethod
        def finalize_response(response) -> None:
            render = getattr(response, 'render', None)
            if callable(render):
                render()

        def response_data(self, response) -> Dict[str, Any]:
            if hasattr(response, 'data'):
                return response.data
            if getattr(response, 'status_code', None) == 204:
                return {}
            try:
                raw = response.content
                if not raw:
                    return {}
                return json.loads(raw.decode())
            except (ValueError, TypeError, UnicodeDecodeError, AttributeError):
                return {}

        def check_http_response(self, response: Response, expected_code: int, expected_message: str = None):
            data = self.response_data(response)
            if expected_message is not None:
                self.assertIn('message', data)
                self.assertEqual(data['message'], expected_message)

                self.assertEqual(len(data.keys()), 1 if expected_code != status.HTTP_400_BAD_REQUEST else 2)

                if expected_code == status.HTTP_400_BAD_REQUEST:
                    self.assertIn('errors', data)
            if response.status_code != expected_code:
                self.fail("Status code {} is different then the expected {}. Response data: {}".format(
                    response.status_code, expected_code, data
                ))

        def get_endpoint_full_uri(self):
            return 'http://localhost:9905{}'.format(self.endpoint)

        def check_single_resource_response(self, request: Request, response_data: Dict, id_field: str = 'id') -> None:
            self.assertIn(id_field, response_data)

            model_object = self.model.objects.get(pk=response_data['id'])
            serializer = self.serializer(model_object, context={'request': request})

            for field in serializer.data:
                self.assertIn(field, response_data)
                self.assertEqual(
                    response_data[field],
                    serializer.data[field],
                    "Content-check failed for field '{}'. Serializer output: ({}) {} - Response output: ({}) {}".format(
                        field,
                        type(serializer.data[field]),
                        serializer.data[field],
                        type(response_data[field]),
                        response_data[field]
                    )
                )

        def check_list_resource_response(
                self,
                request: Request,
                response_data: Dict,
                expected_count: int,
                id_field: str = 'id'
        ) -> None:
            self.assertIn(CustomPagination.KEY_COUNT, response_data)
            self.assertEqual(response_data[CustomPagination.KEY_COUNT], expected_count)

            self.assertIn(CustomPagination.KEY_ITEMS, response_data)
            for item_data in response_data[CustomPagination.KEY_ITEMS]:
                self.check_single_resource_response(request=request, response_data=item_data, id_field=id_field)
