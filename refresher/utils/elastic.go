package utils

import (
	"github.com/infralight/go-kit/db/elasticsearch"
	"github.com/rs/zerolog"
)

func GetK8sIntegrationIds(accountId string, uids []string, kinds []string, client *elasticsearch.Client, logger *zerolog.Logger) ([]string, error) {
		integrationIds, err := client.GetK8sIntegrationIds(accountId, uids, kinds)
	if err != nil {
		logger.Err(err).Msg("failed to get k8s integration ids")
		return nil, err
	}

	return integrationIds, nil

}
