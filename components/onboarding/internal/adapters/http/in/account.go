package in

import (
	libCommons "github.com/LerianStudio/lib-commons/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/commons/opentelemetry"
	libPostgres "github.com/LerianStudio/lib-commons/commons/postgres"
	"github.com/LerianStudio/midaz/components/onboarding/internal/services/command"
	"github.com/LerianStudio/midaz/components/onboarding/internal/services/query"
	"github.com/LerianStudio/midaz/pkg/mmodel"
	"github.com/LerianStudio/midaz/pkg/net/http"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

// AccountHandler struct contains an account use case for managing account related operations.
type AccountHandler struct {
	Command *command.UseCase
	Query   *query.UseCase
}

// CreateAccount is a method that creates account information.
//
//	@Summary		Create an Account
//	@Description	Create an Account with the input payload
//	@Tags			Accounts
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string						true	"Authorization Bearer Token"
//	@Param			X-Request-Id		header		string						false	"Request ID"
//	@Param			organization_id	path		string						true	"Organization ID"
//	@Param			ledger_id		path		string						true	"Ledger ID"
//	@Param			account			body		mmodel.CreateAccountInput	true	"Account"
//	@Success		200				{object}	mmodel.Account
//	@Router			/v1/organizations/{organization_id}/ledgers/{ledger_id}/accounts [post]
func (handler *AccountHandler) CreateAccount(i any, c *fiber.Ctx) error {
	ctx := c.UserContext()

	logger := libCommons.NewLoggerFromContext(ctx)
	tracer := libCommons.NewTracerFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.create_account")
	defer span.End()

	organizationID := c.Locals("organization_id").(uuid.UUID)
	ledgerID := c.Locals("ledger_id").(uuid.UUID)

	payload := i.(*mmodel.CreateAccountInput)
	logger.Infof("Request to create a Account with details: %#v", payload)

	if !libCommons.IsNilOrEmpty(payload.PortfolioID) {
		logger.Infof("Initiating create of Account with Portfolio ID: %s", *payload.PortfolioID)
	}

	err := libOpentelemetry.SetSpanAttributesFromStruct(&span, "payload", payload)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert payload to JSON string", err)

		return http.WithError(c, err)
	}

	account, err := handler.Command.CreateAccount(ctx, organizationID, ledgerID, payload)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to create Account on command", err)

		return http.WithError(c, err)
	}

	logger.Infof("Successfully created Account")

	return http.Created(c, account)
}

// GetAllAccounts is a method that retrieves all Accounts.
//
//	@Summary		Get all Accounts
//	@Description	Get all Accounts with the input metadata or without metadata
//	@Tags			Accounts
//	@Produce		json
//	@Param			Authorization	header		string	true	"Authorization Bearer Token"
//	@Param			X-Request-Id		header		string	false	"Request ID"
//	@Param			organization_id	path		string	true	"Organization ID"
//	@Param			ledger_id		path		string	true	"Ledger ID"
//	@Param			metadata		query		string	false	"Metadata"
//	@Param			limit			query		int		false	"Limit"			default(10)
//	@Param			page			query		int		false	"Page"			default(1)
//	@Param			start_date		query		string	false	"Start Date"	example "2021-01-01"
//	@Param			end_date		query		string	false	"End Date"		example "2021-01-01"
//	@Param			sort_order		query		string	false	"Sort Order"	Enums(asc,desc)
//	@Success		200				{object}	libPostgres.Pagination{items=[]mmodel.Account,page=int,limit=int}
//	@Router			/v1/organizations/{organization_id}/ledgers/{ledger_id}/accounts [get]
func (handler *AccountHandler) GetAllAccounts(c *fiber.Ctx) error {
	ctx := c.UserContext()

	logger := libCommons.NewLoggerFromContext(ctx)
	tracer := libCommons.NewTracerFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.get_all_accounts")
	defer span.End()

	organizationID := c.Locals("organization_id").(uuid.UUID)
	ledgerID := c.Locals("ledger_id").(uuid.UUID)

	var portfolioID *uuid.UUID

	headerParams, err := http.ValidateParameters(c.Queries())
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to validate query parameters", err)
		logger.Errorf("Failed to validate query parameters, Error: %s", err.Error())

		return http.WithError(c, err)
	}

	pagination := libPostgres.Pagination{
		Limit:     headerParams.Limit,
		Page:      headerParams.Page,
		SortOrder: headerParams.SortOrder,
		StartDate: headerParams.StartDate,
		EndDate:   headerParams.EndDate,
	}

	if !libCommons.IsNilOrEmpty(&headerParams.PortfolioID) {
		parsedID := uuid.MustParse(headerParams.PortfolioID)
		portfolioID = &parsedID

		logger.Infof("Search of all Accounts with Portfolio ID: %s", portfolioID)
	}

	if headerParams.Metadata != nil {
		logger.Infof("Initiating retrieval of all Accounts by metadata")

		accounts, err := handler.Query.GetAllMetadataAccounts(ctx, organizationID, ledgerID, portfolioID, *headerParams)
		if err != nil {
			libOpentelemetry.HandleSpanError(&span, "Failed to retrieve all Accounts on query", err)

			logger.Errorf("Failed to retrieve all Accounts, Error: %s", err.Error())

			return http.WithError(c, err)
		}

		logger.Infof("Successfully retrieved all Accounts by metadata")

		pagination.SetItems(accounts)

		return http.OK(c, pagination)
	}

	logger.Infof("Initiating retrieval of all Accounts ")

	headerParams.Metadata = &bson.M{}

	accounts, err := handler.Query.GetAllAccount(ctx, organizationID, ledgerID, portfolioID, *headerParams)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to retrieve all Accounts on query", err)

		logger.Errorf("Failed to retrieve all Accounts, Error: %s", err.Error())

		return http.WithError(c, err)
	}

	logger.Infof("Successfully retrieved all Accounts")

	pagination.SetItems(accounts)

	return http.OK(c, pagination)
}

// GetAccountByID is a method that retrieves Account information by a given account id.
//
//	@Summary		Get an Account by ID
//	@Description	Get an Account with the input ID
//	@Tags			Accounts
//	@Produce		json
//	@Param			Authorization	header		string	true	"Authorization Bearer Token"
//	@Param			X-Request-Id		header		string	false	"Request ID"
//	@Param			organization_id	path		string	true	"Organization ID"
//	@Param			ledger_id		path		string	true	"Ledger ID"
//	@Param			id				path		string	true	"Account ID"
//	@Success		200				{object}	mmodel.Account
//	@Router			/v1/organizations/{organization_id}/ledgers/{ledger_id}/accounts/{id} [get]
func (handler *AccountHandler) GetAccountByID(c *fiber.Ctx) error {
	ctx := c.UserContext()

	logger := libCommons.NewLoggerFromContext(ctx)
	tracer := libCommons.NewTracerFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.get_account_by_id")
	defer span.End()

	organizationID := c.Locals("organization_id").(uuid.UUID)
	ledgerID := c.Locals("ledger_id").(uuid.UUID)
	id := c.Locals("id").(uuid.UUID)

	logger.Infof("Initiating retrieval of Account with Account ID: %s", id.String())

	account, err := handler.Query.GetAccountByID(ctx, organizationID, ledgerID, nil, id)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to retrieve Account on query", err)

		logger.Errorf("Failed to retrieve Account with Account ID: %s, Error: %s", id.String(), err.Error())

		return http.WithError(c, err)
	}

	logger.Infof("Successfully retrieved Account with Account ID: %s", id.String())

	return http.OK(c, account)
}

// GetAccountByAlias is a method that retrieves Account information by a given account alias.
//
//	@Summary		Get an Account by Alias
//	@Description	Get an Account with the input Alias
//	@Tags			Accounts
//	@Produce		json
//	@Param			Authorization	header		string	true	"Authorization Bearer Token"
//	@Param			X-Request-Id		header		string	false	"Request ID"
//	@Param			organization_id	path		string	true	"Organization ID"
//	@Param			ledger_id		path		string	true	"Ledger ID"
//	@Param			alias			path		string	true	"Alias"
//	@Success		200				{object}	mmodel.Account
//	@Router			/v1/organizations/{organization_id}/ledgers/{ledger_id}/accounts/{alias} [get]
func (handler *AccountHandler) GetAccountByAlias(c *fiber.Ctx) error {
	ctx := c.UserContext()

	logger := libCommons.NewLoggerFromContext(ctx)
	tracer := libCommons.NewTracerFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.get_account_by_id")
	defer span.End()

	organizationID := c.Locals("organization_id").(uuid.UUID)
	ledgerID := c.Locals("ledger_id").(uuid.UUID)
	alias := c.Params("alias")

	logger.Infof("Initiating retrieval of Account with Account Alias: %s", alias)

	account, err := handler.Query.GetAccountByAlias(ctx, organizationID, ledgerID, nil, alias)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to retrieve Account on query", err)

		logger.Errorf("Failed to retrieve Account with Account Alias: %s, Error: %s", alias, err.Error())

		return http.WithError(c, err)
	}

	logger.Infof("Successfully retrieved Account with Account Alias: %s", alias)

	return http.OK(c, account)
}

// UpdateAccount is a method that updates Account information.
//
//	@Summary		Update an Account
//	@Description	Update an Account with the input payload
//	@Tags			Accounts
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string						true	"Authorization Bearer Token"
//	@Param			X-Request-Id		header		string						false	"Request ID"
//	@Param			organization_id	path		string						true	"Organization ID"
//	@Param			ledger_id		path		string						true	"Ledger ID"
//	@Param			id				path		string						true	"Account ID"
//	@Param			account			body		mmodel.UpdateAccountInput	true	"Account"
//	@Success		200				{object}	mmodel.Account
//	@Router			/v1/organizations/{organization_id}/ledgers/{ledger_id}/accounts/{id} [patch]
func (handler *AccountHandler) UpdateAccount(i any, c *fiber.Ctx) error {
	ctx := c.UserContext()

	logger := libCommons.NewLoggerFromContext(ctx)
	tracer := libCommons.NewTracerFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.update_account")
	defer span.End()

	organizationID := c.Locals("organization_id").(uuid.UUID)
	ledgerID := c.Locals("ledger_id").(uuid.UUID)
	id := c.Locals("id").(uuid.UUID)

	logger.Infof("Initiating update of Account with ID: %s", id.String())

	payload := i.(*mmodel.UpdateAccountInput)
	logger.Infof("Request to update an Account with details: %#v", payload)

	err := libOpentelemetry.SetSpanAttributesFromStruct(&span, "payload", payload)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert payload to JSON string", err)

		return http.WithError(c, err)
	}

	_, err = handler.Command.UpdateAccount(ctx, organizationID, ledgerID, nil, id, payload)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to update Account on command", err)

		logger.Errorf("Failed to update Account with ID: %s, Error: %s", id.String(), err.Error())

		return http.WithError(c, err)
	}

	account, err := handler.Query.GetAccountByID(ctx, organizationID, ledgerID, nil, id)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to retrieve Account on query", err)

		logger.Errorf("Failed to retrieve Account with ID: %s, Error: %s", id, err.Error())

		return http.WithError(c, err)
	}

	logger.Infof("Successfully updated Account with ID: %s", id.String())

	return http.OK(c, account)
}

// DeleteAccountByID is a method that removes Account information by a given account id.
//
//	@Summary		Delete an Account by ID
//	@Description	Delete an Account with the input ID
//	@Tags			Accounts
//	@Param			Authorization	header	string	true	"Authorization Bearer Token"
//	@Param			X-Request-Id		header	string	false	"Request ID"
//	@Param			organization_id	path	string	true	"Organization ID"
//	@Param			ledger_id		path	string	true	"Ledger ID"
//	@Param			id				path	string	true	"Account ID"
//	@Success		204
//	@Router			/v1/organizations/{organization_id}/ledgers/{ledger_id}/accounts/{id} [delete]
func (handler *AccountHandler) DeleteAccountByID(c *fiber.Ctx) error {
	ctx := c.UserContext()

	logger := libCommons.NewLoggerFromContext(ctx)
	tracer := libCommons.NewTracerFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.delete_account_by_id")
	defer span.End()

	organizationID := c.Locals("organization_id").(uuid.UUID)
	ledgerID := c.Locals("ledger_id").(uuid.UUID)
	id := c.Locals("id").(uuid.UUID)

	logger.Infof("Initiating removal of Account with ID: %s", id.String())

	if err := handler.Command.DeleteAccountByID(ctx, organizationID, ledgerID, nil, id); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to remove Account on command", err)

		logger.Errorf("Failed to remove Account with ID: %s, Error: %s", id.String(), err.Error())

		return http.WithError(c, err)
	}

	logger.Infof("Successfully removed Account with ID: %s", id.String())

	return http.NoContent(c)
}
