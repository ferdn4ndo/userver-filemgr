

class GenericError(Exception):
    pass


class NotFoundError(GenericError):
    pass


class PreConditionFailed(GenericError):
    pass


class UnexpectedCondition(GenericError):
    pass
