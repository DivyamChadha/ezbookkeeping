package api

import (
	"unicode/utf8"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

const maximumTransactionCommentLength = 255

// TransactionSplitHandler splits an existed expense transaction into multiple transactions by request parameters for current user
func (a *TransactionsApi) TransactionSplitHandler(c *core.WebContext) (any, *errs.Error) {
	var transactionSplitReq models.TransactionSplitRequest
	err := c.ShouldBindJSON(&transactionSplitReq)

	if err != nil {
		log.Warnf(c, "[transactions.TransactionSplitHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	clientTimezone, err := c.GetClientTimezone()

	if err != nil {
		log.Warnf(c, "[transactions.TransactionSplitHandler] cannot get client timezone, because %s", err.Error())
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}

	newTransactionsCount := len(transactionSplitReq.ExpenseItems) + len(transactionSplitReq.TransferItems)

	if newTransactionsCount < 1 {
		return nil, errs.ErrTransactionSplitItemsEmpty
	}

	if newTransactionsCount > models.MaximumSplitItemsCountOfTransaction {
		return nil, errs.ErrTransactionSplitHasTooManyItems
	}

	if transactionSplitReq.Amount < 0 {
		return nil, errs.ErrTransactionSplitAmountInvalid
	}

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[transactions.TransactionSplitHandler] failed to get user, because %s", err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	transaction, err := a.transactions.GetTransactionByTransactionId(c, uid, transactionSplitReq.Id)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionSplitHandler] failed to get transaction \"id:%d\" for user \"uid:%d\", because %s", transactionSplitReq.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if transaction.Type != models.TRANSACTION_DB_TYPE_EXPENSE {
		log.Warnf(c, "[transactions.TransactionSplitHandler] cannot split transaction \"id:%d\" for user \"uid:%d\", because transaction type is %d", transactionSplitReq.Id, uid, transaction.Type)
		return nil, errs.ErrTransactionSplitTypeNotSupported
	}

	remainingComment := transaction.Comment

	if transactionSplitReq.Comment != nil {
		remainingComment = *transactionSplitReq.Comment
	}

	newTransactions := make([]*models.Transaction, 0, newTransactionsCount)

	for i := 0; i < len(transactionSplitReq.ExpenseItems); i++ {
		item := transactionSplitReq.ExpenseItems[i]

		if item == nil || item.CategoryId <= 0 {
			return nil, errs.ErrIncompleteOrIncorrectSubmission
		}

		if item.Amount <= 0 || item.Amount > models.MaximumTransactionAmount {
			return nil, errs.ErrTransactionSplitAmountInvalid
		}

		newTransaction, err := a.createSplitTransactionModel(transaction, models.TRANSACTION_DB_TYPE_EXPENSE, item.CategoryId, item.Amount, item.Comment, c.ClientIP())

		if err != nil {
			return nil, err
		}

		newTransactions = append(newTransactions, newTransaction)
	}

	for i := 0; i < len(transactionSplitReq.TransferItems); i++ {
		item := transactionSplitReq.TransferItems[i]

		if item == nil || item.CategoryId <= 0 || item.DestinationAccountId <= 0 {
			return nil, errs.ErrIncompleteOrIncorrectSubmission
		}

		if item.Amount <= 0 || item.Amount > models.MaximumTransactionAmount || item.DestinationAmount < 0 || item.DestinationAmount > models.MaximumTransactionAmount {
			return nil, errs.ErrTransactionSplitAmountInvalid
		}

		if item.DestinationAccountId == transaction.AccountId {
			return nil, errs.ErrTransactionSourceAndDestinationIdCannotBeEqual
		}

		newTransaction, err := a.createSplitTransactionModel(transaction, models.TRANSACTION_DB_TYPE_TRANSFER_OUT, item.CategoryId, item.Amount, item.Comment, c.ClientIP())

		if err != nil {
			return nil, err
		}

		newTransaction.RelatedAccountId = item.DestinationAccountId
		newTransaction.RelatedAccountAmount = item.DestinationAmount

		if item.DestinationAmount == 0 {
			newTransaction.RelatedAccountAmount = item.Amount
		}

		newTransactions = append(newTransactions, newTransaction)
	}

	allUsedAccounts, err := a.getTransactionUsedAccounts(c, uid, append([]*models.Transaction{transaction}, newTransactions...))

	if err != nil {
		log.Errorf(c, "[transactions.TransactionSplitHandler] failed to get transaction used accounts for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	transactionEditable := user.CanEditTransactionByTransactionTime(transaction.TransactionTime, clientTimezone, allUsedAccounts[transaction.AccountId], nil)

	if !transactionEditable {
		return nil, errs.ErrCannotModifyTransactionWithThisTransactionTime
	}

	for i := 0; i < len(newTransactions); i++ {
		newTransactionEditable := user.CanEditTransactionByTransactionTime(transaction.TransactionTime, clientTimezone, allUsedAccounts[transaction.AccountId], allUsedAccounts[newTransactions[i].RelatedAccountId])

		if !newTransactionEditable {
			return nil, errs.ErrCannotCreateTransactionWithThisTransactionTime
		}
	}

	allTransactionTagIds, err := a.transactionTags.GetAllTagIdsOfTransactions(c, uid, []int64{transaction.TransactionId})

	if err != nil {
		log.Errorf(c, "[transactions.TransactionSplitHandler] failed to get transactions tag ids for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	transactionPictureInfos, err := a.transactionPictures.GetPictureInfosByTransactionId(c, uid, transaction.TransactionId)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionSplitHandler] failed to get transaction picture infos for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	modifiedTransaction, newTransactionTagIds, err := a.transactions.SplitTransaction(c, uid, transaction.TransactionId, transactionSplitReq.Amount, transactionSplitReq.CategoryId, remainingComment, newTransactions)

	if err != nil {
		log.Errorf(c, "[transactions.TransactionSplitHandler] failed to split transaction \"id:%d\" for user \"uid:%d\", because %s", transactionSplitReq.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[transactions.TransactionSplitHandler] user \"uid:%d\" has split transaction \"id:%d\" into %d new transactions successfully", uid, transactionSplitReq.Id, len(newTransactions))

	transactionSplitResp := &models.TransactionSplitResponse{
		Transaction:     modifiedTransaction.ToTransactionInfoResponse(allTransactionTagIds[transaction.TransactionId], transactionEditable),
		NewTransactions: make([]*models.TransactionInfoResponse, len(newTransactions)),
	}

	transactionSplitResp.Transaction.Pictures = a.GetTransactionPictureInfoResponseList(transactionPictureInfos)

	for i := 0; i < len(newTransactions); i++ {
		transactionSplitResp.NewTransactions[i] = newTransactions[i].ToTransactionInfoResponse(newTransactionTagIds, true)
	}

	return transactionSplitResp, nil
}

func (a *TransactionsApi) createSplitTransactionModel(transaction *models.Transaction, transactionDbType models.TransactionDbType, categoryId int64, amount int64, comment *string, clientIp string) (*models.Transaction, *errs.Error) {
	newTransaction := &models.Transaction{
		Uid:               transaction.Uid,
		Type:              transactionDbType,
		CategoryId:        categoryId,
		TransactionTime:   transaction.TransactionTime,
		TimezoneUtcOffset: transaction.TimezoneUtcOffset,
		AccountId:         transaction.AccountId,
		Amount:            amount,
		HideAmount:        transaction.HideAmount,
		Comment:           transaction.Comment,
		GeoLongitude:      transaction.GeoLongitude,
		GeoLatitude:       transaction.GeoLatitude,
		CreatedIp:         clientIp,
	}

	if comment != nil {
		if utf8.RuneCountInString(*comment) > maximumTransactionCommentLength {
			return nil, errs.ErrIncompleteOrIncorrectSubmission
		}

		newTransaction.Comment = *comment
	}

	return newTransaction, nil
}
