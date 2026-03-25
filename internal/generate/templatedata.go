package generate

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ishuar/tfskel/internal/strutil"
	"github.com/ishuar/tfskel/internal/templates"
)

// prepareTemplateData extracts config values and builds template data
func (g *Generator) prepareTemplateData(env, region, appDir string) (*templates.Data, error) {
	shortRegion := strutil.TransformRegionName(region)

	// Extract nested config values with nil checks
	awsProviderVersion := "~> 6.0"
	defaultTags := make(map[string]string)
	s3BucketName := "CHANGE_ME_WITH_YOUR_GLOBALLY_UNIQUE_S3_BUCKET_NAME"

	if g.config.Provider != nil && g.config.Provider.AWS != nil {
		if g.config.Provider.AWS.Version != "" {
			awsProviderVersion = g.config.Provider.AWS.Version
		}
		if g.config.Provider.AWS.DefaultTags != nil {
			// Normalize tag keys to lowercase for Terraform compatibility
			// This is needed when WRITING to files to ensure consistent tag format
			// (matches the normalization in extractMetadata for comparison)
			for k, v := range g.config.Provider.AWS.DefaultTags {
				defaultTags[strings.ToLower(k)] = v
			}
		}
	}

	if g.config.Backend != nil && g.config.Backend.S3 != nil && g.config.Backend.S3.BucketName != "" {
		s3BucketName = g.config.Backend.S3.BucketName
	}

	// Get account ID for the environment
	accountID, err := g.config.GetAccountID(env)
	if err != nil {
		return nil, err
	}

	// Create initial data for template rendering
	data := &templates.Data{
		Env:                env,
		Region:             region,
		AppDir:             appDir,
		AccountID:          accountID,
		ShortRegion:        shortRegion,
		S3BucketName:       s3BucketName,
		TerraformVersion:   g.config.TerraformVersion,
		AWSProviderVersion: awsProviderVersion,
		DefaultTags:        defaultTags,
	}

	// Build AWS role ARN for terraform workflows (needs full context for templates)
	awsRoleArn, err := g.buildAWSRoleArn(data)
	if err != nil {
		return nil, fmt.Errorf("failed to build AWS role ARN: %w", err)
	}
	data.AWSRoleArn = awsRoleArn

	// Render bucket_name as a template if it contains Go template syntax
	renderedBucketName, err := g.renderer.RenderConfigValue(s3BucketName, "bucket_name", data)
	if err != nil {
		return nil, fmt.Errorf("failed to render bucket_name template: %w", err)
	}
	data.S3BucketName = renderedBucketName

	return data, nil
}

// buildAWSRoleArn constructs AWS role ARN from config or returns explicit ARN
// Priority: aws_role_arn > aws_role_name > default placeholder
// Accepts full template data to allow flexible template variables in role names/ARNs
func (g *Generator) buildAWSRoleArn(data *templates.Data) (string, error) {
	// Defensive: check Workflows is initialized
	if g.config.Workflows == nil {
		// Return default placeholder
		return fmt.Sprintf("arn:aws:iam::%s:role/REPLACE_WITH_ROLE_TO_ASSUME", data.AccountID), nil
	}

	workflows := g.config.Workflows

	// Create template data once for reuse in both ARN and role name rendering
	// This provides all available fields for maximum flexibility in templates
	templateData := &templates.Data{
		Env:         data.Env,
		Region:      data.Region,
		AppDir:      data.AppDir,
		AccountID:   data.AccountID,
		ShortRegion: data.ShortRegion,
	}

	// If explicit ARN is provided, render it as a template and use it
	if workflows.AWSRoleArn != "" {
		renderedArn, err := g.renderer.RenderConfigValue(workflows.AWSRoleArn, "aws_role_arn", templateData)
		if err != nil {
			return "", fmt.Errorf("invalid template syntax in aws_role_arn: %w", err)
		}
		return renderedArn, nil
	}

	// If role name is provided, render it as a template and construct ARN
	if workflows.AWSRoleName != "" {
		renderedRoleName, err := g.renderer.RenderConfigValue(workflows.AWSRoleName, "aws_role_name", templateData)
		if err != nil {
			return "", fmt.Errorf("invalid template syntax in aws_role_name: %w", err)
		}
		return fmt.Sprintf("arn:aws:iam::%s:role/%s", data.AccountID, renderedRoleName), nil
	}

	// Return default placeholder
	return fmt.Sprintf("arn:aws:iam::%s:role/REPLACE_WITH_ROLE_TO_ASSUME", data.AccountID), nil
}

// enrichTemplateDataForWorkflow adds workflow-specific data to template context
// For github workflow templates, computes and injects the workflow filename
func (g *Generator) enrichTemplateDataForWorkflow(tmplPath string, data *templates.Data) error {
	normalizedPath := filepath.ToSlash(tmplPath)
	parts := strings.Split(normalizedPath, "/")

	if len(parts) == 0 {
		return nil
	}

	// Only process github workflow templates (.tmpl files)
	if parts[0] != categoryGithub || !strings.HasSuffix(tmplPath, ".tmpl") {
		return nil
	}

	// Extract the original filename
	fileName := parts[len(parts)-1]
	fileName = strings.TrimSuffix(fileName, ".tmpl")

	// Skip reusable workflows
	if strings.HasPrefix(fileName, "reusable-") {
		return nil
	}

	// Generate and inject workflow filename
	workflowFileName, err := g.generateWorkflowFileName(fileName, data)
	if err != nil {
		return fmt.Errorf("failed to generate workflow filename: %w", err)
	}

	data.WorkflowFileName = workflowFileName
	return nil
}
