# Terraform custom resources

Terraform custom provider for resources with non-standard options and flags.

dashsoftaws_dynamodb_table has the key: only_scale_up

The flag (when set to true) prevents Terraform from scaling down tables that have had their read_capacity or
write_capacity turned up from outside Terraform (for instance by operators in response to production workloads).

