package services

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
	"github.com/mayswind/ezbookkeeping/pkg/uuid"
)

const (
	splitTestUid = int64(1)

	splitTestBankAccountId       = int64(1001)
	splitTestReceivableAccountId = int64(1002)
	splitTestHiddenAccountId     = int64(1003)

	splitTestFacilityCategoryId  = int64(2001)
	splitTestGroceriesCategoryId = int64(2002)
	splitTestSnacksCategoryId    = int64(2003)
	splitTestIncomeCategoryId    = int64(2011)
	splitTestTransferCategoryId  = int64(2021)

	splitTestVisibleTagId = int64(3001)
	splitTestHiddenTagId  = int64(3002)

	splitTestBankInitialBalance = int64(10000000)
	splitTestTransactionTime    = int64(1790000000)
)

func setupSplitTestDatabase(t *testing.T) core.Context {
	config := &settings.Config{
		DatabaseConfig: &settings.DatabaseConfig{
			DatabaseType: settings.Sqlite3DbType,
			DatabasePath: filepath.Join(t.TempDir(), "ezbookkeeping_test.db"),
		},
		UuidGeneratorType: settings.InternalUuidGeneratorType,
	}

	assert.Nil(t, datastore.InitializeDataStore(config))
	assert.Nil(t, uuid.InitializeUuidGenerator(config))

	assert.Nil(t, datastore.Container.UserDataStore.SyncStructs(new(models.Account)))
	assert.Nil(t, datastore.Container.UserDataStore.SyncStructs(new(models.Transaction)))
	assert.Nil(t, datastore.Container.UserDataStore.SyncStructs(new(models.TransactionCategory)))
	assert.Nil(t, datastore.Container.UserDataStore.SyncStructs(new(models.TransactionTag)))
	assert.Nil(t, datastore.Container.UserDataStore.SyncStructs(new(models.TransactionTagIndex)))
	assert.Nil(t, datastore.Container.UserDataStore.SyncStructs(new(models.TransactionPictureInfo)))

	c := core.NewNullContext()
	sess := Transactions.UserDataDB(splitTestUid).NewSession(c)

	accounts := []*models.Account{
		{AccountId: splitTestBankAccountId, Uid: splitTestUid, Category: models.ACCOUNT_CATEGORY_SAVINGS_ACCOUNT, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Name: "Bank", Currency: "INR", Balance: splitTestBankInitialBalance},
		{AccountId: splitTestReceivableAccountId, Uid: splitTestUid, Category: models.ACCOUNT_CATEGORY_RECEIVABLES, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Name: "Due back", Currency: "INR"},
		{AccountId: splitTestHiddenAccountId, Uid: splitTestUid, Category: models.ACCOUNT_CATEGORY_RECEIVABLES, Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Name: "Hidden", Currency: "INR", Hidden: true},
	}

	categories := []*models.TransactionCategory{
		{CategoryId: 2000, Uid: splitTestUid, Type: models.CATEGORY_TYPE_EXPENSE, Name: "Expense"},
		{CategoryId: splitTestFacilityCategoryId, Uid: splitTestUid, Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: 2000, Name: "Facility"},
		{CategoryId: splitTestGroceriesCategoryId, Uid: splitTestUid, Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: 2000, Name: "Groceries"},
		{CategoryId: splitTestSnacksCategoryId, Uid: splitTestUid, Type: models.CATEGORY_TYPE_EXPENSE, ParentCategoryId: 2000, Name: "Snacks"},
		{CategoryId: 2010, Uid: splitTestUid, Type: models.CATEGORY_TYPE_INCOME, Name: "Income"},
		{CategoryId: splitTestIncomeCategoryId, Uid: splitTestUid, Type: models.CATEGORY_TYPE_INCOME, ParentCategoryId: 2010, Name: "Uncategorized"},
		{CategoryId: 2020, Uid: splitTestUid, Type: models.CATEGORY_TYPE_TRANSFER, Name: "Transfer"},
		{CategoryId: splitTestTransferCategoryId, Uid: splitTestUid, Type: models.CATEGORY_TYPE_TRANSFER, ParentCategoryId: 2020, Name: "Uncategorized"},
	}

	tags := []*models.TransactionTag{
		{TagId: splitTestVisibleTagId, Uid: splitTestUid, Name: "Swiggy"},
		{TagId: splitTestHiddenTagId, Uid: splitTestUid, Name: "Old", Hidden: true},
	}

	for i := 0; i < len(accounts); i++ {
		_, err := sess.Insert(accounts[i])
		assert.Nil(t, err)
	}

	for i := 0; i < len(categories); i++ {
		_, err := sess.Insert(categories[i])
		assert.Nil(t, err)
	}

	for i := 0; i < len(tags); i++ {
		_, err := sess.Insert(tags[i])
		assert.Nil(t, err)
	}

	return c
}

func createSplitTestTransaction(t *testing.T, c core.Context, transactionType models.TransactionDbType, categoryId int64, amount int64, comment string, tagIds []int64) *models.Transaction {
	transaction := &models.Transaction{
		Uid:               splitTestUid,
		Type:              transactionType,
		CategoryId:        categoryId,
		TransactionTime:   utils.GetMinTransactionTimeFromUnixTime(splitTestTransactionTime),
		TimezoneUtcOffset: 330,
		AccountId:         splitTestBankAccountId,
		Amount:            amount,
		Comment:           comment,
	}

	err := Transactions.CreateTransaction(c, transaction, tagIds, nil)
	assert.Nil(t, err)

	return transaction
}

func getSplitTestAccountBalance(t *testing.T, c core.Context, accountId int64) int64 {
	account := &models.Account{}
	has, err := Transactions.UserDataDB(splitTestUid).NewSession(c).ID(accountId).Get(account)

	assert.Nil(t, err)
	assert.True(t, has)

	return account.Balance
}

func getSplitTestTransactions(t *testing.T, c core.Context) []*models.Transaction {
	var transactions []*models.Transaction
	err := Transactions.UserDataDB(splitTestUid).NewSession(c).Where("uid=? AND deleted=?", splitTestUid, false).OrderBy("transaction_time asc").Find(&transactions)

	assert.Nil(t, err)

	return transactions
}

func getSplitTestTransactionTagIds(t *testing.T, c core.Context, transactionId int64) []int64 {
	var tagIndexes []*models.TransactionTagIndex
	err := Transactions.UserDataDB(splitTestUid).NewSession(c).Where("uid=? AND deleted=? AND transaction_id=?", splitTestUid, false, transactionId).OrderBy("tag_id asc").Find(&tagIndexes)

	assert.Nil(t, err)

	tagIds := make([]int64, len(tagIndexes))

	for i := 0; i < len(tagIndexes); i++ {
		tagIds[i] = tagIndexes[i].TagId
	}

	return tagIds
}

func TestSplitTransaction_ByCategories(t *testing.T) {
	c := setupSplitTestDatabase(t)
	transaction := createSplitTestTransaction(t, c, models.TRANSACTION_DB_TYPE_EXPENSE, splitTestGroceriesCategoryId, 100000, "online order", []int64{splitTestVisibleTagId})

	assert.Equal(t, splitTestBankInitialBalance-100000, getSplitTestAccountBalance(t, c, splitTestBankAccountId))

	newTransactions := []*models.Transaction{
		{Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: splitTestSnacksCategoryId, Amount: 30000, Comment: "online order"},
	}

	modifiedTransaction, tagIds, err := Transactions.SplitTransaction(c, splitTestUid, transaction.TransactionId, 70000, splitTestGroceriesCategoryId, "online order", newTransactions)

	assert.Nil(t, err)
	assert.Equal(t, []int64{splitTestVisibleTagId}, tagIds)
	assert.Equal(t, int64(70000), modifiedTransaction.Amount)

	transactions := getSplitTestTransactions(t, c)
	assert.Equal(t, 2, len(transactions))

	assert.Equal(t, transaction.TransactionId, transactions[0].TransactionId)
	assert.Equal(t, models.TRANSACTION_DB_TYPE_EXPENSE, transactions[0].Type)
	assert.Equal(t, splitTestGroceriesCategoryId, transactions[0].CategoryId)
	assert.Equal(t, int64(70000), transactions[0].Amount)
	assert.Equal(t, "online order", transactions[0].Comment)

	assert.Equal(t, newTransactions[0].TransactionId, transactions[1].TransactionId)
	assert.Equal(t, models.TRANSACTION_DB_TYPE_EXPENSE, transactions[1].Type)
	assert.Equal(t, splitTestSnacksCategoryId, transactions[1].CategoryId)
	assert.Equal(t, splitTestBankAccountId, transactions[1].AccountId)
	assert.Equal(t, int64(30000), transactions[1].Amount)
	assert.Equal(t, int64(0), transactions[1].RelatedAccountId)
	assert.Equal(t, int16(330), transactions[1].TimezoneUtcOffset)
	assert.Equal(t, "online order", transactions[1].Comment)
	assert.Equal(t, splitTestTransactionTime, utils.GetUnixTimeFromTransactionTime(transactions[1].TransactionTime))
	assert.NotEqual(t, transactions[0].TransactionTime, transactions[1].TransactionTime)

	assert.Equal(t, []int64{splitTestVisibleTagId}, getSplitTestTransactionTagIds(t, c, transactions[0].TransactionId))
	assert.Equal(t, []int64{splitTestVisibleTagId}, getSplitTestTransactionTagIds(t, c, transactions[1].TransactionId))

	assert.Equal(t, splitTestBankInitialBalance-100000, getSplitTestAccountBalance(t, c, splitTestBankAccountId))
}

func TestSplitTransaction_ByCategoriesAndChangeRemainingCategoryAndComment(t *testing.T) {
	c := setupSplitTestDatabase(t)
	transaction := createSplitTestTransaction(t, c, models.TRANSACTION_DB_TYPE_EXPENSE, splitTestFacilityCategoryId, 100000, "online order", nil)

	newTransactions := []*models.Transaction{
		{Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: splitTestSnacksCategoryId, Amount: 25000},
		{Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: splitTestFacilityCategoryId, Amount: 15000},
	}

	_, tagIds, err := Transactions.SplitTransaction(c, splitTestUid, transaction.TransactionId, 60000, splitTestGroceriesCategoryId, "groceries part", newTransactions)

	assert.Nil(t, err)
	assert.Equal(t, 0, len(tagIds))

	transactions := getSplitTestTransactions(t, c)
	assert.Equal(t, 3, len(transactions))

	assert.Equal(t, splitTestGroceriesCategoryId, transactions[0].CategoryId)
	assert.Equal(t, int64(60000), transactions[0].Amount)
	assert.Equal(t, "groceries part", transactions[0].Comment)
	assert.Equal(t, splitTestSnacksCategoryId, transactions[1].CategoryId)
	assert.Equal(t, int64(25000), transactions[1].Amount)
	assert.Equal(t, splitTestFacilityCategoryId, transactions[2].CategoryId)
	assert.Equal(t, int64(15000), transactions[2].Amount)

	assert.Equal(t, splitTestBankInitialBalance-100000, getSplitTestAccountBalance(t, c, splitTestBankAccountId))
}

func TestSplitTransaction_WithPeople(t *testing.T) {
	c := setupSplitTestDatabase(t)
	transaction := createSplitTestTransaction(t, c, models.TRANSACTION_DB_TYPE_EXPENSE, splitTestFacilityCategoryId, 108378, "Playo", []int64{splitTestVisibleTagId})

	newTransactions := []*models.Transaction{
		{Type: models.TRANSACTION_DB_TYPE_TRANSFER_OUT, CategoryId: splitTestTransferCategoryId, Amount: 72200, RelatedAccountId: splitTestReceivableAccountId, RelatedAccountAmount: 72200, Comment: "Playo"},
	}

	modifiedTransaction, _, err := Transactions.SplitTransaction(c, splitTestUid, transaction.TransactionId, 36178, splitTestFacilityCategoryId, "Playo", newTransactions)

	assert.Nil(t, err)
	assert.Equal(t, int64(36178), modifiedTransaction.Amount)

	transactions := getSplitTestTransactions(t, c)
	assert.Equal(t, 3, len(transactions))

	assert.Equal(t, models.TRANSACTION_DB_TYPE_EXPENSE, transactions[0].Type)
	assert.Equal(t, int64(36178), transactions[0].Amount)

	assert.Equal(t, models.TRANSACTION_DB_TYPE_TRANSFER_OUT, transactions[1].Type)
	assert.Equal(t, splitTestTransferCategoryId, transactions[1].CategoryId)
	assert.Equal(t, splitTestBankAccountId, transactions[1].AccountId)
	assert.Equal(t, int64(72200), transactions[1].Amount)
	assert.Equal(t, splitTestReceivableAccountId, transactions[1].RelatedAccountId)
	assert.Equal(t, int64(72200), transactions[1].RelatedAccountAmount)
	assert.Equal(t, transactions[2].TransactionId, transactions[1].RelatedId)

	assert.Equal(t, models.TRANSACTION_DB_TYPE_TRANSFER_IN, transactions[2].Type)
	assert.Equal(t, splitTestReceivableAccountId, transactions[2].AccountId)
	assert.Equal(t, int64(72200), transactions[2].Amount)
	assert.Equal(t, splitTestBankAccountId, transactions[2].RelatedAccountId)
	assert.Equal(t, transactions[1].TransactionId, transactions[2].RelatedId)
	assert.Equal(t, transactions[1].TransactionTime+1, transactions[2].TransactionTime)

	assert.Equal(t, splitTestBankInitialBalance-108378, getSplitTestAccountBalance(t, c, splitTestBankAccountId))
	assert.Equal(t, int64(72200), getSplitTestAccountBalance(t, c, splitTestReceivableAccountId))
}

func TestSplitTransaction_ByCategoriesAndWithPeopleAtSameTime(t *testing.T) {
	c := setupSplitTestDatabase(t)
	transaction := createSplitTestTransaction(t, c, models.TRANSACTION_DB_TYPE_EXPENSE, splitTestGroceriesCategoryId, 100000, "", nil)

	newTransactions := []*models.Transaction{
		{Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: splitTestSnacksCategoryId, Amount: 20000},
		{Type: models.TRANSACTION_DB_TYPE_TRANSFER_OUT, CategoryId: splitTestTransferCategoryId, Amount: 50000, RelatedAccountId: splitTestReceivableAccountId, RelatedAccountAmount: 50000},
		{Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: splitTestFacilityCategoryId, Amount: 30000},
	}

	_, _, err := Transactions.SplitTransaction(c, splitTestUid, transaction.TransactionId, 0, splitTestGroceriesCategoryId, "", newTransactions)

	assert.Nil(t, err)

	transactions := getSplitTestTransactions(t, c)
	assert.Equal(t, 5, len(transactions))
	assert.Equal(t, int64(0), transactions[0].Amount)

	transactionTimes := make(map[int64]bool)

	for i := 0; i < len(transactions); i++ {
		assert.Equal(t, splitTestTransactionTime, utils.GetUnixTimeFromTransactionTime(transactions[i].TransactionTime))
		transactionTimes[transactions[i].TransactionTime] = true
	}

	assert.Equal(t, 5, len(transactionTimes))

	assert.Equal(t, splitTestBankInitialBalance-100000, getSplitTestAccountBalance(t, c, splitTestBankAccountId))
	assert.Equal(t, int64(50000), getSplitTestAccountBalance(t, c, splitTestReceivableAccountId))
}

func TestSplitTransaction_HiddenTagIsNotCopied(t *testing.T) {
	c := setupSplitTestDatabase(t)
	transaction := createSplitTestTransaction(t, c, models.TRANSACTION_DB_TYPE_EXPENSE, splitTestGroceriesCategoryId, 100000, "", []int64{splitTestVisibleTagId})

	_, err := Transactions.UserDataDB(splitTestUid).NewSession(c).Insert(&models.TransactionTagIndex{
		TagIndexId:      9001,
		Uid:             splitTestUid,
		TagId:           splitTestHiddenTagId,
		TransactionId:   transaction.TransactionId,
		TransactionTime: transaction.TransactionTime,
	})
	assert.Nil(t, err)

	newTransactions := []*models.Transaction{
		{Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: splitTestSnacksCategoryId, Amount: 30000},
	}

	_, tagIds, err := Transactions.SplitTransaction(c, splitTestUid, transaction.TransactionId, 70000, splitTestGroceriesCategoryId, "", newTransactions)

	assert.Nil(t, err)
	assert.Equal(t, []int64{splitTestVisibleTagId}, tagIds)
	assert.Equal(t, []int64{splitTestVisibleTagId, splitTestHiddenTagId}, getSplitTestTransactionTagIds(t, c, transaction.TransactionId))
	assert.Equal(t, []int64{splitTestVisibleTagId}, getSplitTestTransactionTagIds(t, c, newTransactions[0].TransactionId))
}

func TestSplitTransaction_AmountNotEqual(t *testing.T) {
	c := setupSplitTestDatabase(t)
	transaction := createSplitTestTransaction(t, c, models.TRANSACTION_DB_TYPE_EXPENSE, splitTestGroceriesCategoryId, 100000, "", nil)

	newTransactions := []*models.Transaction{
		{Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: splitTestSnacksCategoryId, Amount: 30000},
	}

	_, _, err := Transactions.SplitTransaction(c, splitTestUid, transaction.TransactionId, 60000, splitTestGroceriesCategoryId, "", newTransactions)

	assert.Equal(t, errs.ErrTransactionSplitAmountNotEqual, err)

	transactions := getSplitTestTransactions(t, c)
	assert.Equal(t, 1, len(transactions))
	assert.Equal(t, int64(100000), transactions[0].Amount)
	assert.Equal(t, splitTestBankInitialBalance-100000, getSplitTestAccountBalance(t, c, splitTestBankAccountId))
}

func TestSplitTransaction_InvalidArguments(t *testing.T) {
	c := setupSplitTestDatabase(t)
	transaction := createSplitTestTransaction(t, c, models.TRANSACTION_DB_TYPE_EXPENSE, splitTestGroceriesCategoryId, 100000, "", nil)

	_, _, err := Transactions.SplitTransaction(c, splitTestUid, transaction.TransactionId, 100000, splitTestGroceriesCategoryId, "", nil)
	assert.Equal(t, errs.ErrTransactionSplitItemsEmpty, err)

	tooManyTransactions := make([]*models.Transaction, models.MaximumSplitItemsCountOfTransaction+1)

	for i := 0; i < len(tooManyTransactions); i++ {
		tooManyTransactions[i] = &models.Transaction{Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: splitTestSnacksCategoryId, Amount: 1}
	}

	_, _, err = Transactions.SplitTransaction(c, splitTestUid, transaction.TransactionId, 100000-int64(len(tooManyTransactions)), splitTestGroceriesCategoryId, "", tooManyTransactions)
	assert.Equal(t, errs.ErrTransactionSplitHasTooManyItems, err)

	_, _, err = Transactions.SplitTransaction(c, splitTestUid, transaction.TransactionId, -10000, splitTestGroceriesCategoryId, "", []*models.Transaction{
		{Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: splitTestSnacksCategoryId, Amount: 110000},
	})
	assert.Equal(t, errs.ErrTransactionSplitAmountInvalid, err)

	_, _, err = Transactions.SplitTransaction(c, splitTestUid, transaction.TransactionId, 100000, splitTestGroceriesCategoryId, "", []*models.Transaction{
		{Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: splitTestSnacksCategoryId, Amount: 0},
	})
	assert.Equal(t, errs.ErrTransactionSplitAmountInvalid, err)

	_, _, err = Transactions.SplitTransaction(c, splitTestUid, transaction.TransactionId, 70000, splitTestGroceriesCategoryId, "", []*models.Transaction{
		{Type: models.TRANSACTION_DB_TYPE_INCOME, CategoryId: splitTestIncomeCategoryId, Amount: 30000},
	})
	assert.Equal(t, errs.ErrTransactionTypeInvalid, err)

	_, _, err = Transactions.SplitTransaction(c, splitTestUid, 123456789, 70000, splitTestGroceriesCategoryId, "", []*models.Transaction{
		{Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: splitTestSnacksCategoryId, Amount: 30000},
	})
	assert.Equal(t, errs.ErrTransactionNotFound, err)

	_, _, err = Transactions.SplitTransaction(c, splitTestUid+1, transaction.TransactionId, 70000, splitTestGroceriesCategoryId, "", []*models.Transaction{
		{Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: splitTestSnacksCategoryId, Amount: 30000},
	})
	assert.Equal(t, errs.ErrTransactionNotFound, err)

	transactions := getSplitTestTransactions(t, c)
	assert.Equal(t, 1, len(transactions))
	assert.Equal(t, int64(100000), transactions[0].Amount)
}

func TestSplitTransaction_OnlyExpenseCanBeSplit(t *testing.T) {
	c := setupSplitTestDatabase(t)
	transaction := createSplitTestTransaction(t, c, models.TRANSACTION_DB_TYPE_INCOME, splitTestIncomeCategoryId, 100000, "", nil)

	_, _, err := Transactions.SplitTransaction(c, splitTestUid, transaction.TransactionId, 70000, splitTestIncomeCategoryId, "", []*models.Transaction{
		{Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: splitTestSnacksCategoryId, Amount: 30000},
	})

	assert.Equal(t, errs.ErrTransactionSplitTypeNotSupported, err)
	assert.Equal(t, splitTestBankInitialBalance+100000, getSplitTestAccountBalance(t, c, splitTestBankAccountId))
}

func TestSplitTransaction_NothingChangedWhenAnyItemFailed(t *testing.T) {
	c := setupSplitTestDatabase(t)
	transaction := createSplitTestTransaction(t, c, models.TRANSACTION_DB_TYPE_EXPENSE, splitTestGroceriesCategoryId, 100000, "online order", []int64{splitTestVisibleTagId})

	// the category of the last item does not exist
	_, _, err := Transactions.SplitTransaction(c, splitTestUid, transaction.TransactionId, 40000, splitTestFacilityCategoryId, "changed", []*models.Transaction{
		{Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: splitTestSnacksCategoryId, Amount: 10000},
		{Type: models.TRANSACTION_DB_TYPE_TRANSFER_OUT, CategoryId: splitTestTransferCategoryId, Amount: 20000, RelatedAccountId: splitTestReceivableAccountId, RelatedAccountAmount: 20000},
		{Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: 987654321, Amount: 30000},
	})
	assert.Equal(t, errs.ErrTransactionCategoryNotFound, err)

	// the destination account is hidden
	_, _, err = Transactions.SplitTransaction(c, splitTestUid, transaction.TransactionId, 40000, splitTestGroceriesCategoryId, "online order", []*models.Transaction{
		{Type: models.TRANSACTION_DB_TYPE_TRANSFER_OUT, CategoryId: splitTestTransferCategoryId, Amount: 60000, RelatedAccountId: splitTestHiddenAccountId, RelatedAccountAmount: 60000},
	})
	assert.Equal(t, errs.ErrCannotAddTransactionToHiddenAccount, err)

	// the destination account is the account of the transaction itself
	_, _, err = Transactions.SplitTransaction(c, splitTestUid, transaction.TransactionId, 40000, splitTestGroceriesCategoryId, "online order", []*models.Transaction{
		{Type: models.TRANSACTION_DB_TYPE_TRANSFER_OUT, CategoryId: splitTestTransferCategoryId, Amount: 60000, RelatedAccountId: splitTestBankAccountId, RelatedAccountAmount: 60000},
	})
	assert.Equal(t, errs.ErrTransactionSourceAndDestinationIdCannotBeEqual, err)

	// the category type of the item is not matched with the transaction type
	_, _, err = Transactions.SplitTransaction(c, splitTestUid, transaction.TransactionId, 40000, splitTestGroceriesCategoryId, "online order", []*models.Transaction{
		{Type: models.TRANSACTION_DB_TYPE_EXPENSE, CategoryId: splitTestTransferCategoryId, Amount: 60000},
	})
	assert.Equal(t, errs.ErrTransactionCategoryTypeInvalid, err)

	// the amounts of the transfer in the same currency are not equal
	_, _, err = Transactions.SplitTransaction(c, splitTestUid, transaction.TransactionId, 40000, splitTestGroceriesCategoryId, "online order", []*models.Transaction{
		{Type: models.TRANSACTION_DB_TYPE_TRANSFER_OUT, CategoryId: splitTestTransferCategoryId, Amount: 60000, RelatedAccountId: splitTestReceivableAccountId, RelatedAccountAmount: 50000},
	})
	assert.Equal(t, errs.ErrTransactionSourceAndDestinationAmountNotEqual, err)

	transactions := getSplitTestTransactions(t, c)
	assert.Equal(t, 1, len(transactions))
	assert.Equal(t, splitTestGroceriesCategoryId, transactions[0].CategoryId)
	assert.Equal(t, int64(100000), transactions[0].Amount)
	assert.Equal(t, "online order", transactions[0].Comment)

	assert.Equal(t, splitTestBankInitialBalance-100000, getSplitTestAccountBalance(t, c, splitTestBankAccountId))
	assert.Equal(t, int64(0), getSplitTestAccountBalance(t, c, splitTestReceivableAccountId))
}

// The following tests cover the behaviors of modifying transaction which "Mark as repayment" and "Mark as refund" depend on

func TestModifyTransaction_MarkIncomeAsRepaymentFromReceivableAccount(t *testing.T) {
	c := setupSplitTestDatabase(t)
	transaction := createSplitTestTransaction(t, c, models.TRANSACTION_DB_TYPE_EXPENSE, splitTestFacilityCategoryId, 108378, "Playo", nil)

	_, _, err := Transactions.SplitTransaction(c, splitTestUid, transaction.TransactionId, 36178, splitTestFacilityCategoryId, "Playo", []*models.Transaction{
		{Type: models.TRANSACTION_DB_TYPE_TRANSFER_OUT, CategoryId: splitTestTransferCategoryId, Amount: 72200, RelatedAccountId: splitTestReceivableAccountId, RelatedAccountAmount: 72200},
	})
	assert.Nil(t, err)

	repayment := &models.Transaction{
		Uid:               splitTestUid,
		Type:              models.TRANSACTION_DB_TYPE_INCOME,
		CategoryId:        splitTestIncomeCategoryId,
		TransactionTime:   utils.GetMinTransactionTimeFromUnixTime(splitTestTransactionTime + 86400),
		TimezoneUtcOffset: 330,
		AccountId:         splitTestBankAccountId,
		Amount:            36100,
		Comment:           "credited from friend",
	}
	assert.Nil(t, Transactions.CreateTransaction(c, repayment, nil, nil))
	assert.Equal(t, splitTestBankInitialBalance-108378+36100, getSplitTestAccountBalance(t, c, splitTestBankAccountId))

	err = Transactions.ModifyTransaction(c, &models.Transaction{
		TransactionId:        repayment.TransactionId,
		Uid:                  splitTestUid,
		Type:                 models.TRANSACTION_DB_TYPE_TRANSFER_OUT,
		CategoryId:           splitTestTransferCategoryId,
		TransactionTime:      utils.GetMinTransactionTimeFromUnixTime(splitTestTransactionTime + 86400),
		TimezoneUtcOffset:    330,
		AccountId:            splitTestReceivableAccountId,
		Amount:               36100,
		RelatedAccountId:     splitTestBankAccountId,
		RelatedAccountAmount: 36100,
		Comment:              "credited from friend",
	}, true, 0, nil, nil, nil, nil)
	assert.Nil(t, err)

	modifiedRepayment, err := Transactions.GetTransactionByTransactionId(c, splitTestUid, repayment.TransactionId)
	assert.Nil(t, err)
	assert.Equal(t, models.TRANSACTION_DB_TYPE_TRANSFER_OUT, modifiedRepayment.Type)
	assert.Equal(t, splitTestReceivableAccountId, modifiedRepayment.AccountId)
	assert.Equal(t, splitTestBankAccountId, modifiedRepayment.RelatedAccountId)
	assert.Equal(t, int64(36100), modifiedRepayment.Amount)
	assert.Equal(t, int64(36100), modifiedRepayment.RelatedAccountAmount)

	relatedTransaction, err := Transactions.GetTransactionByTransactionId(c, splitTestUid, modifiedRepayment.RelatedId)
	assert.Nil(t, err)
	assert.Equal(t, models.TRANSACTION_DB_TYPE_TRANSFER_IN, relatedTransaction.Type)
	assert.Equal(t, splitTestBankAccountId, relatedTransaction.AccountId)
	assert.Equal(t, int64(36100), relatedTransaction.Amount)

	incomeCount, err := Transactions.UserDataDB(splitTestUid).NewSession(c).Where("uid=? AND deleted=? AND type=?", splitTestUid, false, models.TRANSACTION_DB_TYPE_INCOME).Count(&models.Transaction{})
	assert.Nil(t, err)
	assert.Equal(t, int64(0), incomeCount)

	assert.Equal(t, splitTestBankInitialBalance-108378+36100, getSplitTestAccountBalance(t, c, splitTestBankAccountId))
	assert.Equal(t, int64(72200-36100), getSplitTestAccountBalance(t, c, splitTestReceivableAccountId))
}

func TestModifyTransaction_MarkIncomeAsRefundOfExpense(t *testing.T) {
	c := setupSplitTestDatabase(t)
	createSplitTestTransaction(t, c, models.TRANSACTION_DB_TYPE_EXPENSE, splitTestSnacksCategoryId, 300000, "sent by mistake", nil)

	refund := &models.Transaction{
		Uid:               splitTestUid,
		Type:              models.TRANSACTION_DB_TYPE_INCOME,
		CategoryId:        splitTestIncomeCategoryId,
		TransactionTime:   utils.GetMinTransactionTimeFromUnixTime(splitTestTransactionTime + 180),
		TimezoneUtcOffset: 330,
		AccountId:         splitTestBankAccountId,
		Amount:            270000,
		Comment:           "refunded",
	}
	assert.Nil(t, Transactions.CreateTransaction(c, refund, nil, nil))
	assert.Equal(t, splitTestBankInitialBalance-300000+270000, getSplitTestAccountBalance(t, c, splitTestBankAccountId))

	err := Transactions.ModifyTransaction(c, &models.Transaction{
		TransactionId:     refund.TransactionId,
		Uid:               splitTestUid,
		Type:              models.TRANSACTION_DB_TYPE_EXPENSE,
		CategoryId:        splitTestSnacksCategoryId,
		TransactionTime:   utils.GetMinTransactionTimeFromUnixTime(splitTestTransactionTime + 180),
		TimezoneUtcOffset: 330,
		AccountId:         splitTestBankAccountId,
		Amount:            -270000,
		Comment:           "refunded",
	}, false, 0, nil, nil, nil, nil)
	assert.Nil(t, err)

	modifiedRefund, err := Transactions.GetTransactionByTransactionId(c, splitTestUid, refund.TransactionId)
	assert.Nil(t, err)
	assert.Equal(t, models.TRANSACTION_DB_TYPE_EXPENSE, modifiedRefund.Type)
	assert.Equal(t, splitTestSnacksCategoryId, modifiedRefund.CategoryId)
	assert.Equal(t, int64(-270000), modifiedRefund.Amount)

	totalExpense, err := Transactions.UserDataDB(splitTestUid).NewSession(c).Where("uid=? AND deleted=? AND type=? AND category_id=?", splitTestUid, false, models.TRANSACTION_DB_TYPE_EXPENSE, splitTestSnacksCategoryId).SumInt(&models.Transaction{}, "amount")
	assert.Nil(t, err)
	assert.Equal(t, int64(30000), totalExpense)

	assert.Equal(t, splitTestBankInitialBalance-300000+270000, getSplitTestAccountBalance(t, c, splitTestBankAccountId))
}
