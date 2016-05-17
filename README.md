# Terraform custom resources

Terraform custom provider for resources with non-standard options and flags.

dashsoftaws_ecs_cluster: Sets remaining services (if any) to desired count 0 and deletes them (to be able to delete a
cluster where services have been placed programatically)

dashsoftaws_dynamodb_table has the key: only_scale_up

dashsoftaws_iam_group: will remove all manually added users before deleting the group

The flag (when set to true) prevents Terraform from scaling down tables that have had their read_capacity or
write_capacity turned up from outside Terraform (for instance by operators in response to production workloads).

dashsoftaws_api_gwateway_deployment: has more keys (cachecluster, clientcertificateid, burst- and ratelimit et. al.)

dashsoftaws_api_gateaway_base_path_mapping
dashsoftaws_api_gateaway_client_certificate
dashsoftaws_api_gateaway_base_path_mapping
dashsoftaws_api_gateaway_domain_name
dashsoftaws_kms_grant

API Gateway and KMS resources that are not in the official Terraform at the moment

Build: go build -o $GOPATH/bin/terraform-provider-dashsoftaws