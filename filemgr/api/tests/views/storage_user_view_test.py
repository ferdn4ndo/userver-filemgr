import unittest
import uuid
from unittest.mock import patch

from api.serializers.storage.storage_user_serializer import StorageUserSerializer
from core.services.auth.u_server_authentication_service import UServerAuthenticationService
from api.tests import dataset
from api.tests import generic
from api.views.storage.storage_user_view import StorageUserViewSet
from core.models import StorageUser


class StorageUserViewSetCreateTest(generic.GenericCreateTestCases.CreateTest):
    serializer = StorageUserSerializer
    model = StorageUser
    requires_admin = True
    view = StorageUserViewSet

    def setUp(self) -> None:
        super().setUp()
        self.storage = dataset.create_storage()
        self.user1 = dataset.create_user()
        self._auth_resolve_patch = patch.object(
            UServerAuthenticationService,
            'get_user_from_system_name_and_username',
            return_value=self.user1,
        )
        self._auth_resolve_patch.start()
        self.payload_valid = {
            'username': self.user1.username,
            'system_name': self.user1.system_name,
            'may_write': True,
            'may_read': True,
        }
        self.payload_invalid = {
            'username': self.user1.username,
            'system_name': self.user1.system_name,
            'may_write': 1234,
            'may_read': 4553,
        }
        self.endpoint = '/storages/{}/users/'.format(self.storage.id)
        self.request_args = {'storage_id': self.storage.id}

    def tearDown(self) -> None:
        self._auth_resolve_patch.stop()
        if self.user1 is not None:
            self.user1.delete()
        if self.storage is not None:
            self.storage.delete()
        super().tearDown()


class StorageUserViewSetDestroyTest(generic.GenericDestroyTestCase.DestroyTest):
    serializer = StorageUserSerializer
    model = StorageUser
    requires_admin = True
    view = StorageUserViewSet
    model_fake_id = uuid.uuid4()

    def setUp(self) -> None:
        super().setUp()
        self.storage = dataset.create_storage()
        self.user1 = dataset.create_user()
        self.endpoint = '/storages/{}/users/'.format(self.storage.id)
        self.request_args = {'storage_id': self.storage.id}
        self.storage_user = dataset.create_storage_user(storage=self.storage, user=self.user1, creator=self.user)
        self.model_real_id = self.storage_user.id

    def tearDown(self) -> None:
        if self.storage_user is not None:
            self.storage_user.delete()
        if self.user1 is not None:
            self.user1.delete()
        if self.storage is not None:
            self.storage.delete()
        super().tearDown()


class StorageUserViewSetListTest(generic.GenericListTestCase.ListTest):
    serializer = StorageUserSerializer
    model = StorageUser
    requires_admin = True
    view = StorageUserViewSet
    expected_items = 20

    def setUp(self) -> None:
        super().setUp()
        self.storage = dataset.create_storage()
        self.storage_users = dataset.create_storage_users(storage=self.storage, total_users=20, creator=self.user)
        self.endpoint = '/storages/{}/users/'.format(self.storage.id)
        self.request_args = {'storage_id': self.storage.id}

    def tearDown(self) -> None:
        for user in self.storage_users:
            user.delete()
        if self.storage is not None:
            self.storage.delete()
        super().tearDown()


@unittest.skip('StorageUserViewSet does not implement partial_update (create/read/destroy only).')
class StorageUserViewSetPartialUpdateTest(generic.GenericPartialUpdateTestCase.PartialUpdateTest):
    serializer = StorageUserSerializer
    model = StorageUser
    requires_admin = True
    view = StorageUserViewSet
    model_fake_id = uuid.uuid4()

    def setUp(self) -> None:
        super().setUp()
        self.storage = dataset.create_storage()
        self.user1 = dataset.create_user()
        self.payload_valid = {
            'username': self.user1.username,
            'system_name': self.user1.system_name,
            'may_write': True,
            'may_read': True,
        }
        self.payload_invalid = {
            'username': self.user1.username,
            'system_name': self.user1.system_name,
            'may_write': 1234,
            'may_read': 4553,
        }
        self.endpoint = '/storages/{}/users/'.format(self.storage.id)
        self.request_args = {'storage_id': self.storage.id}
        self.storage_user = dataset.create_storage_user(storage=self.storage, user=self.user1, creator=self.user)
        self.model_real_id = self.storage_user.id

    def tearDown(self) -> None:
        if self.user1 is not None:
            self.user1.delete()
        if self.storage is not None:
            self.storage.delete()
        super().tearDown()


class StorageUserViewSetRetrieveTest(generic.GenericRetrieveTestCase.RetrieveTest):
    serializer = StorageUserSerializer
    model = StorageUser
    requires_admin = True
    view = StorageUserViewSet
    model_fake_id = uuid.uuid4()

    def setUp(self) -> None:
        super().setUp()
        self.storage = dataset.create_storage()
        self.user1 = dataset.create_user()
        self.endpoint = '/storages/{}/users/'.format(self.storage.id)
        self.request_args = {'storage_id': self.storage.id}
        self.storage_user = dataset.create_storage_user(storage=self.storage, user=self.user1, creator=self.user)
        self.model_real_id = self.storage_user.id

    def tearDown(self) -> None:
        if self.user1 is not None:
            self.user1.delete()
        if self.storage is not None:
            self.storage.delete()
        super().tearDown()
