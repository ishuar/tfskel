package drift

// DefaultCriticalResources returns the default list of AWS resources considered critical.
// These resources are categorized by their risk level when modified or deleted.
func DefaultCriticalResources() []string {
	return []string{
		// Database resources - risk of data loss or service disruption
		"aws_db_instance",
		"aws_db_cluster",
		"aws_rds_cluster",
		"aws_rds_cluster_instance",
		"aws_rds_global_cluster",
		"aws_db_subnet_group",
		"aws_db_parameter_group",
		"aws_rds_cluster_parameter_group",
		"aws_dynamodb_table",
		"aws_dynamodb_global_table",
		"aws_elasticache_cluster",
		"aws_elasticache_replication_group",
		"aws_redshift_cluster",
		"aws_neptune_cluster",
		"aws_neptune_cluster_instance",
		"aws_docdb_cluster",
		"aws_docdb_cluster_instance",

		// Storage resources - risk of data loss
		"aws_s3_bucket",
		"aws_s3_bucket_policy",
		"aws_s3_bucket_public_access_block",
		"aws_efs_file_system",
		"aws_efs_access_point",
		"aws_backup_vault",
		"aws_backup_plan",

		// Network resources - risk of service disruption
		"aws_vpc",
		"aws_subnet",
		"aws_route_table",
		"aws_route_table_association",
		"aws_internet_gateway",
		"aws_nat_gateway",
		"aws_vpc_peering_connection",
		"aws_vpn_gateway",
		"aws_vpn_connection",
		"aws_customer_gateway",
		"aws_transit_gateway",
		"aws_transit_gateway_route_table",
		"aws_vpc_endpoint",
		"aws_vpc_endpoint_service",

		// Security resources - risk of unauthorized access or security breach
		"aws_security_group",
		"aws_security_group_rule",
		"aws_network_acl",
		"aws_network_acl_rule",
		"aws_iam_role",
		"aws_iam_role_policy",
		"aws_iam_role_policy_attachment",
		"aws_iam_policy",
		"aws_iam_user",
		"aws_iam_user_policy",
		"aws_iam_user_policy_attachment",
		"aws_iam_group",
		"aws_iam_group_policy",
		"aws_iam_group_policy_attachment",
		"aws_kms_key",
		"aws_kms_alias",
		"aws_secretsmanager_secret",
		"aws_secretsmanager_secret_version",

		// WAF resources - risk of exposing applications to attacks
		"aws_waf_web_acl",
		"aws_waf_rule",
		"aws_waf_rule_group",
		"aws_wafv2_web_acl",
		"aws_wafv2_rule_group",
		"aws_wafv2_ip_set",
		"aws_wafv2_regex_pattern_set",
		"aws_waf_rate_based_rule",
		"aws_shield_protection",
		"aws_shield_protection_group",
	}
}

// MergeCriticalResources merges default and user-defined critical resources,
// removing duplicates and maintaining order (defaults first, then user-defined).
func MergeCriticalResources(defaults, userDefined []string) []string {
	seen := make(map[string]bool, len(defaults)+len(userDefined))
	merged := make([]string, 0, len(defaults)+len(userDefined))

	// Add defaults first
	for _, resource := range defaults {
		if !seen[resource] {
			merged = append(merged, resource)
			seen[resource] = true
		}
	}

	// Add user-defined resources
	for _, resource := range userDefined {
		if resource != "" && !seen[resource] {
			merged = append(merged, resource)
			seen[resource] = true
		}
	}

	return merged
}
