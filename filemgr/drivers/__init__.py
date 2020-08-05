from filemgr.drivers.amazon_s3 import AmazonS3Driver
from filemgr.drivers.generic import GenericDriver
from filemgr.drivers.local import LocalDriver
from filemgr.models.storage_file_model import StorageFile
from filemgr.models.storage_model import Storage


def load_storage_driver(storage: Storage) -> [GenericDriver]:
    if storage.type == Storage.StorageType.STORAGE_S3:
        return AmazonS3Driver(storage)
    elif storage.type == Storage.StorageType.STORAGE_LOCAL:
        return LocalDriver(storage)
    return None


def load_storage_file_driver(file: StorageFile) -> [GenericDriver]:
    """
    Retrieves the appropriate storage driver for a given file
    :return:
    """
    return load_storage_driver(file.storage)
