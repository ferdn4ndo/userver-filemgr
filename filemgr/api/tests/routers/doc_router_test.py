import os

from django.test import Client
from rest_framework import status

from ..generic import GenericTestCase


class DocsRouterOpenAPITest(GenericTestCase.GenericTest):

    def test_openapi_return_success_not_logged(self) -> None:
        host = os.environ.get('VIRTUAL_HOST', 'localhost')
        client = Client(HTTP_HOST=host)
        response = client.get('/docs/openapi/')
        self.assertEqual(response.status_code, status.HTTP_200_OK)
        content_type = response.get('Content-Type', '')
        self.assertIn('application/vnd.oai.openapi', content_type)


class DocsRouterRedocTest(GenericTestCase.GenericTest):

    def test_redoc_return_success_not_logged(self) -> None:
        host = os.environ.get('VIRTUAL_HOST', 'localhost')
        client = Client(HTTP_HOST=host)
        response = client.get('/docs/redoc/')
        self.assertEqual(response.status_code, status.HTTP_200_OK)
        content_type = response.get('Content-Type', '')
        self.assertIn('text/html', content_type)
        self.assertIn('<title>ReDoc</title>', response.content.decode())
