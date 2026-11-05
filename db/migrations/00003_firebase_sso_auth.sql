-- +goose Up
-- +goose StatementBegin
ALTER TABLE account_identities
  DROP CONSTRAINT account_identities_provider_check;
ALTER TABLE account_identities
  ADD CONSTRAINT account_identities_provider_check
  CHECK (provider IN ('google','apple','email_otp','firebase'));

ALTER TABLE admin_users
  ADD COLUMN account_id uuid REFERENCES accounts(id);
CREATE UNIQUE INDEX uq_admin_users_account
  ON admin_users (account_id)
  WHERE account_id IS NOT NULL AND deleted_at IS NULL;

ALTER TABLE refresh_tokens
  ADD COLUMN client text NOT NULL DEFAULT 'guest_mobile'
  CHECK (client IN ('guest_mobile','fleet_console'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE refresh_tokens DROP COLUMN client;

DROP INDEX IF EXISTS uq_admin_users_account;
ALTER TABLE admin_users DROP COLUMN account_id;

ALTER TABLE account_identities
  DROP CONSTRAINT account_identities_provider_check;
ALTER TABLE account_identities
  ADD CONSTRAINT account_identities_provider_check
  CHECK (provider IN ('google','apple','email_otp'));
-- +goose StatementEnd
