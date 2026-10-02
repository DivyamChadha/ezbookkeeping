package services

import (
	"time"

	"xorm.io/xorm"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
	"github.com/mayswind/ezbookkeeping/pkg/uuid"
)

// SplitTransaction keeps a part of the amount in an existed expense transaction and saves the rest as new transactions
// in the same account and at the same time, all changes are made in one database transaction.
// Each new transaction should be an expense transaction or a transfer (out) transaction, the caller sets its type,
// category, amount, destination account, comment and so on, the other fields are set from the existed transaction.
// It returns the modified existed transaction and the tag ids which are copied to the new transactions.
func (s *TransactionService) SplitTransaction(c core.Context, uid int64, transactionId int64, remainingAmount int64, remainingCategoryId int64, remainingComment string, newTransactions []*models.Transaction) (*models.Transaction, []int64, error) {
	if uid <= 0 {
		return nil, nil, errs.ErrUserIdInvalid
	}

	if len(newTransactions) < 1 {
		return nil, nil, errs.ErrTransactionSplitItemsEmpty
	}

	if len(newTransactions) > models.MaximumSplitItemsCountOfTransaction {
		return nil, nil, errs.ErrTransactionSplitHasTooManyItems
	}

	if remainingAmount < 0 {
		return nil, nil, errs.ErrTransactionSplitAmountInvalid
	}

	totalAmount := remainingAmount
	needTransactionUuidCount := 0

	for i := 0; i < len(newTransactions); i++ {
		newTransaction := newTransactions[i]

		if newTransaction.Type == models.TRANSACTION_DB_TYPE_EXPENSE {
			needTransactionUuidCount++
		} else if newTransaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			needTransactionUuidCount += 2
		} else {
			return nil, nil, errs.ErrTransactionTypeInvalid
		}

		if newTransaction.Amount <= 0 || newTransaction.Amount > models.MaximumTransactionAmount {
			return nil, nil, errs.ErrTransactionSplitAmountInvalid
		}

		totalAmount += newTransaction.Amount
	}

	transactionUuids := s.GenerateUuids(uuid.UUID_TYPE_TRANSACTION, uint16(needTransactionUuidCount))

	if len(transactionUuids) < needTransactionUuidCount {
		return nil, nil, errs.ErrSystemIsBusy
	}

	now := time.Now().Unix()
	userDataDb := s.UserDataDB(uid)

	var transaction *models.Transaction
	var tagIds []int64

	err := userDataDb.DoTransaction(c, func(sess *xorm.Session) error {
		// Get and verify current transaction
		oldTransaction := &models.Transaction{}
		has, err := sess.ID(transactionId).Where("uid=? AND deleted=?", uid, false).Get(oldTransaction)

		if err != nil {
			log.Errorf(c, "[transactions.SplitTransaction] failed to get current transaction, because %s", err.Error())
			return err
		} else if !has {
			return errs.ErrTransactionNotFound
		}

		if oldTransaction.Type != models.TRANSACTION_DB_TYPE_EXPENSE {
			return errs.ErrTransactionSplitTypeNotSupported
		}

		if oldTransaction.Amount <= 0 {
			return errs.ErrTransactionSplitAmountInvalid
		}

		if totalAmount != oldTransaction.Amount {
			return errs.ErrTransactionSplitAmountNotEqual
		}

		// Get and verify account
		account := &models.Account{}
		has, err = sess.ID(oldTransaction.AccountId).Where("uid=? AND deleted=?", uid, false).Get(account)

		if err != nil {
			log.Errorf(c, "[transactions.SplitTransaction] failed to get account, because %s", err.Error())
			return err
		} else if !has {
			return errs.ErrSourceAccountNotFound
		}

		if account.Hidden {
			return errs.ErrCannotModifyTransactionInHiddenAccount
		}

		// Update current transaction row
		modifiedTransaction := *oldTransaction
		modifiedTransaction.Amount = remainingAmount
		modifiedTransaction.CategoryId = remainingCategoryId
		modifiedTransaction.Comment = remainingComment
		modifiedTransaction.UpdatedUnixTime = now

		updateCols := []string{"amount", "updated_unix_time"}

		if modifiedTransaction.CategoryId != oldTransaction.CategoryId {
			err = s.isCategoryValid(sess, &modifiedTransaction)

			if err != nil {
				return err
			}

			updateCols = append(updateCols, "category_id")
		}

		if modifiedTransaction.Comment != oldTransaction.Comment {
			updateCols = append(updateCols, "comment")
		}

		updatedRows, err := sess.ID(modifiedTransaction.TransactionId).Cols(updateCols...).Where("uid=? AND deleted=?", uid, false).Update(&modifiedTransaction)

		if err != nil {
			log.Errorf(c, "[transactions.SplitTransaction] failed to update transaction, because %s", err.Error())
			return err
		} else if updatedRows < 1 {
			return errs.ErrTransactionNotFound
		}

		// Give the amount which is no longer in current transaction back to the account, the new transactions would take it again
		account.UpdatedUnixTime = now
		updatedRows, err = s.updateAccountBalance(sess, account, oldTransaction.Amount-remainingAmount)

		if err != nil {
			log.Errorf(c, "[transactions.SplitTransaction] failed to update account balance, because %s", err.Error())
			return err
		} else if updatedRows < 1 {
			log.Errorf(c, "[transactions.SplitTransaction] failed to update account balance")
			return errs.ErrDatabaseOperationFailed
		}

		// Get the visible tags of current transaction, these tags would be copied to the new transactions
		tagIds, err = s.getVisibleTagIdsOfTransaction(sess, uid, oldTransaction.TransactionId)

		if err != nil {
			log.Errorf(c, "[transactions.SplitTransaction] failed to get transaction tags, because %s", err.Error())
			return err
		}

		needTagIndexUuidCount := len(tagIds) * len(newTransactions)
		tagIndexUuids := s.GenerateUuids(uuid.UUID_TYPE_TAG_INDEX, uint16(needTagIndexUuidCount))

		if len(tagIndexUuids) < needTagIndexUuidCount {
			return errs.ErrSystemIsBusy
		}

		// Get the next available transaction time in the same second of current transaction
		sameSecondLatestTransaction := &models.Transaction{}
		minTransactionTime := utils.GetMinTransactionTimeFromUnixTime(utils.GetUnixTimeFromTransactionTime(oldTransaction.TransactionTime))
		maxTransactionTime := utils.GetMaxTransactionTimeFromUnixTime(utils.GetUnixTimeFromTransactionTime(oldTransaction.TransactionTime))

		has, err = sess.Where("uid=? AND transaction_time>=? AND transaction_time<=?", uid, minTransactionTime, maxTransactionTime).OrderBy("transaction_time desc").Limit(1).Get(sameSecondLatestTransaction)

		if err != nil {
			log.Errorf(c, "[transactions.SplitTransaction] failed to get trasaction time, because %s", err.Error())
			return err
		} else if !has {
			log.Errorf(c, "[transactions.SplitTransaction] it should have transactions in %d - %d, but result is empty", minTransactionTime, maxTransactionTime)
			return errs.ErrDatabaseOperationFailed
		}

		nextTransactionTime := sameSecondLatestTransaction.TransactionTime + 1
		transactionUuidIndex := 0

		// Create new transactions
		for i := 0; i < len(newTransactions); i++ {
			newTransaction := newTransactions[i]
			needTransactionTimeCount := int64(1)

			if newTransaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
				needTransactionTimeCount = 2
			}

			if nextTransactionTime+needTransactionTimeCount-1 > maxTransactionTime-1 {
				return errs.ErrTooMuchTransactionInOneSecond
			}

			newTransaction.TransactionId = transactionUuids[transactionUuidIndex]
			transactionUuidIndex++

			if newTransaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
				newTransaction.RelatedId = transactionUuids[transactionUuidIndex]
				transactionUuidIndex++
			}

			newTransaction.Uid = uid
			newTransaction.Deleted = false
			newTransaction.AccountId = oldTransaction.AccountId
			newTransaction.TransactionTime = nextTransactionTime
			newTransaction.TimezoneUtcOffset = oldTransaction.TimezoneUtcOffset
			newTransaction.ScheduledCreated = false
			newTransaction.CreatedUnixTime = now
			newTransaction.UpdatedUnixTime = now

			err = s.isAccountIdValid(newTransaction)

			if err != nil {
				return err
			}

			transactionTagIndexes := make([]*models.TransactionTagIndex, len(tagIds))

			for j := 0; j < len(tagIds); j++ {
				transactionTagIndexes[j] = &models.TransactionTagIndex{
					TagIndexId:      tagIndexUuids[i*len(tagIds)+j],
					Uid:             uid,
					Deleted:         false,
					TagId:           tagIds[j],
					TransactionId:   newTransaction.TransactionId,
					CreatedUnixTime: now,
					UpdatedUnixTime: now,
				}
			}

			err = s.doCreateTransaction(c, userDataDb, sess, newTransaction, transactionTagIndexes, tagIds, nil, nil)

			if err != nil {
				return err
			}

			nextTransactionTime = newTransaction.TransactionTime + needTransactionTimeCount
		}

		transaction = &modifiedTransaction

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return transaction, tagIds, nil
}

func (s *TransactionService) getVisibleTagIdsOfTransaction(sess *xorm.Session, uid int64, transactionId int64) ([]int64, error) {
	var tagIndexes []*models.TransactionTagIndex
	err := sess.Where("uid=? AND deleted=? AND transaction_id=?", uid, false, transactionId).OrderBy("tag_index_id asc").Find(&tagIndexes)

	if err != nil {
		return nil, err
	}

	if len(tagIndexes) < 1 {
		return make([]int64, 0), nil
	}

	allTagIds := make([]int64, len(tagIndexes))

	for i := 0; i < len(tagIndexes); i++ {
		allTagIds[i] = tagIndexes[i].TagId
	}

	allTagIds = utils.ToUniqueInt64Slice(allTagIds)

	var tags []*models.TransactionTag
	err = sess.Where("uid=? AND deleted=? AND hidden=?", uid, false, false).In("tag_id", allTagIds).Find(&tags)

	if err != nil {
		return nil, err
	}

	visibleTagIds := make(map[int64]bool, len(tags))

	for i := 0; i < len(tags); i++ {
		visibleTagIds[tags[i].TagId] = true
	}

	tagIds := make([]int64, 0, len(tags))

	for i := 0; i < len(allTagIds); i++ {
		if visibleTagIds[allTagIds[i]] {
			tagIds = append(tagIds, allTagIds[i])
		}
	}

	return tagIds, nil
}
