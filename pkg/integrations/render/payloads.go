package render

func serviceDataFromService(service Service) map[string]any {
	data := map[string]any{
		"serviceId":   service.ID,
		"serviceName": service.Name,
		"suspended":   service.Suspended,
	}

	if service.OwnerID != "" {
		data["ownerId"] = service.OwnerID
	}
	if service.Type != "" {
		data["type"] = service.Type
	}
	if service.CreatedAt != "" {
		data["createdAt"] = service.CreatedAt
	}
	if service.UpdatedAt != "" {
		data["updatedAt"] = service.UpdatedAt
	}
	if service.DashboardURL != "" {
		data["dashboardUrl"] = service.DashboardURL
	}
	if service.Slug != "" {
		data["slug"] = service.Slug
	}
	if service.RootDir != "" {
		data["rootDir"] = service.RootDir
	}
	if len(service.Suspenders) > 0 {
		data["suspenders"] = service.Suspenders
	}
	if service.AutoDeploy != "" {
		data["autoDeploy"] = service.AutoDeploy
	}
	if service.NotifyOnFail != "" {
		data["notifyOnFail"] = service.NotifyOnFail
	}
	if service.Repo != "" {
		data["repo"] = service.Repo
	}
	if service.Branch != "" {
		data["branch"] = service.Branch
	}
	if service.EnvironmentID != "" {
		data["environmentId"] = service.EnvironmentID
	}
	if service.ImagePath != "" {
		data["imagePath"] = service.ImagePath
	}
	if len(service.ServiceDetails) > 0 {
		data["serviceDetails"] = service.ServiceDetails
	}

	return data
}

func deployDataFromDeployResponse(serviceID string, deploy DeployResponse) map[string]any {
	data := map[string]any{
		"deployId":  deploy.ID,
		"serviceId": serviceID,
	}

	if deploy.Status != "" {
		data["status"] = deploy.Status
	}
	if deploy.Trigger != "" {
		data["trigger"] = deploy.Trigger
	}
	if deploy.CreatedAt != "" {
		data["createdAt"] = deploy.CreatedAt
	}
	if deploy.UpdatedAt != "" {
		data["updatedAt"] = deploy.UpdatedAt
	}
	if deploy.StartedAt != "" {
		data["startedAt"] = deploy.StartedAt
	}
	if deploy.FinishedAt != "" {
		data["finishedAt"] = deploy.FinishedAt
	}

	if deploy.Commit != nil {
		commit := map[string]any{}
		if deploy.Commit.ID != "" {
			commit["id"] = deploy.Commit.ID
		}
		if deploy.Commit.Message != "" {
			commit["message"] = deploy.Commit.Message
		}
		if deploy.Commit.CreatedAt != "" {
			commit["createdAt"] = deploy.Commit.CreatedAt
		}
		if len(commit) > 0 {
			data["commit"] = commit
		}
	}

	if deploy.Image != nil {
		image := map[string]any{}
		if deploy.Image.Ref != "" {
			image["ref"] = deploy.Image.Ref
		}
		if deploy.Image.SHA != "" {
			image["sha"] = deploy.Image.SHA
		}
		if deploy.Image.RegistryCredential != "" {
			image["registryCredential"] = deploy.Image.RegistryCredential
		}
		if len(image) > 0 {
			data["image"] = image
		}
	}

	return data
}
