package errs

import "net/http"

// Error codes related to splitting transactions
// The indexes start from 900 to avoid conflicting with the error codes defined in transaction.go
var (
	ErrTransactionSplitItemsEmpty       = NewNormalError(NormalSubcategoryTransaction, 900, http.StatusBadRequest, "transaction split items cannot be empty")
	ErrTransactionSplitHasTooManyItems  = NewNormalError(NormalSubcategoryTransaction, 901, http.StatusBadRequest, "transaction split has too many items")
	ErrTransactionSplitAmountInvalid    = NewNormalError(NormalSubcategoryTransaction, 902, http.StatusBadRequest, "transaction split amount is invalid")
	ErrTransactionSplitAmountNotEqual   = NewNormalError(NormalSubcategoryTransaction, 903, http.StatusBadRequest, "total amount of split items is not equal to transaction amount")
	ErrTransactionSplitTypeNotSupported = NewNormalError(NormalSubcategoryTransaction, 904, http.StatusBadRequest, "only expense transaction can be split")
)
