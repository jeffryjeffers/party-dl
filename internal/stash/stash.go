package stash

import (
	"context"
	"fmt"
	"github.com/machinebox/graphql"
)

type UpdateInput struct {
	ID          float64 `json:"id,omitempty"`
	Title       string  `json:"title,omitempty"`
	Date        string  `json:"date,omitempty"`
	StudioID    string  `json:"studio_id,omitempty"`
	Details     string  `json:"details,omitempty"`
	URLs        string  `json:"urls,omitempty"`
	PerformerID string  `json:"performer_ids,omitempty"`
}

type Manager struct {
	Client *graphql.Client
	URL    string
}

func NewManager(url string) *Manager {
	client := graphql.NewClient(url + "/graphql")
	return &Manager{
		Client: client,
		URL:    url,
	}
}

func (s *Manager) findEntity(query string, variables map[string]interface{}, findKey string, responseKey string) (string, error) {
	response, err := s.executeQuery(query, variables, findKey)
	if err != nil {
		return "", err
	}

	count := int(response["count"].(float64))
	if count == 0 {
		return "", nil
	}
	return response[responseKey].([]interface{})[0].(map[string]interface{})["id"].(string), nil
}

func (s *Manager) GetOrCreateStudio(name, url string) (string, error) {
	findQuery := `
		query FindStudios($name: String!) {
		  findStudios(filter: {q: ""}, studio_filter: {name: {value: $name, modifier: EQUALS}}) {
		    count
		    studios {
		      id
		      name
		    }
		  }
		}
	`

	variables := map[string]interface{}{
		"name": name,
	}
	entityID, err := s.findEntity(findQuery, variables, "findStudios", "studios")
	if err != nil {
		return "", err
	}

	if entityID == "" {
		return s.createEntity("studioCreate", "StudioCreateInput", map[string]string{"name": name, "url": url})
	}

	return entityID, nil
}

func (s *Manager) GetOrCreatePerformer(performerName string, url string) (string, error) {
	findQuery := `
		query FindPerformers($filter: FindFilterType, $performer_filter: PerformerFilterType) {
			findPerformers(filter: $filter, performer_filter: $performer_filter) {
				count
				performers {
					id
					name
				}
			}
		}
	`

	variables := map[string]interface{}{
		"filter": map[string]string{"q": performerName},
	}
	entityID, err := s.findEntity(findQuery, variables, "findPerformers", "performers")
	if err != nil {
		return "", err
	}

	if entityID == "" {
		return s.createEntity("performerCreate", "PerformerCreateInput", map[string]string{"name": performerName, "url": url})
	}

	return entityID, nil
}

func (s *Manager) GetSceneByPathAndSize(path string, fileSize int64) ([]interface{}, bool, error) {
	query := fmt.Sprintf(`
        mutation {
            querySQL(
                sql: "SELECT folders.path, files.basename, files.size, files.id AS files_id, folders.id AS folders_id, scenes.id AS scenes_id, scenes.title AS scenes_title, scenes.details AS scenes_details FROM files JOIN folders ON files.parent_folder_id = folders.id JOIN scenes_files ON files.id = scenes_files.file_id JOIN scenes ON scenes.id = scenes_files.scene_id  WHERE files.basename LIKE '%%%s%%' AND files.size = %d"
            ) {
                rows
            }
        }
    `, path, fileSize)

	variables := map[string]interface{}{}

	responseData, err := s.executeQuery(query, variables, "querySQL")
	if err != nil {
		return nil, false, err
	}

	if len(responseData["rows"].([]interface{})) == 0 {
		return nil, false, nil
	}

	return responseData["rows"].([]interface{})[0].([]interface{}), true, nil
}

func (s *Manager) GetImageByPathAndSize(path string, fileSize int64) ([]interface{}, bool, error) {
	query := fmt.Sprintf(`
		mutation {
			querySQL(
				sql: "SELECT folders.path, files.basename, files.size, files.id AS files_id, folders.id AS folders_id, images.id AS images_id, images.title AS images_title FROM files JOIN folders ON files.parent_folder_id=folders.id JOIN images_files ON files.id = images_files.file_id JOIN images ON images.id = images_files.image_id WHERE files.basename LIKE '%%%s%%' AND files.size = %d"
			) {
				rows
			}
		}
	`, path, fileSize)

	variables := map[string]interface{}{}

	responseData, err := s.executeQuery(query, variables, "querySQL")
	if err != nil {
		return nil, false, err
	}

	if len(responseData["rows"].([]interface{})) == 0 {
		return nil, false, nil
	}

	return responseData["rows"].([]interface{})[0].([]interface{}), true, nil
}

func (s *Manager) UpdateScene(sceneUpdateInput UpdateInput) (map[string]interface{}, error) {
	mutation := `
        mutation sceneUpdate($sceneUpdateInput: SceneUpdateInput!){
            sceneUpdate(input: $sceneUpdateInput){
                id
                title
                date
                studio {
                    id
                }
				performers {
					id
				}
                details
                urls
            }
        }
    `

	variables := map[string]interface{}{
		"sceneUpdateInput": sceneUpdateInput,
	}

	return s.executeMutation(mutation, variables)
}

func (s *Manager) UpdateImage(imageUpdateInput UpdateInput) (map[string]interface{}, error) {
	mutation := `
        mutation imageUpdate($imageUpdateInput: ImageUpdateInput!){
            imageUpdate(input: $imageUpdateInput){
                id
                title
                date
                studio {
                    id
                }
				performers {
					id
				}
                details
                urls
            }
        }
    `

	variables := map[string]interface{}{
		"imageUpdateInput": imageUpdateInput,
	}

	return s.executeMutation(mutation, variables)
}

func (s *Manager) createEntity(mutationName, mutationType string, input map[string]string) (string, error) {
	mutation := fmt.Sprintf(`
		mutation ($input: %s!) {
			%s(input: $input) {
				id
			}
		}
	`, mutationType, mutationName)
	mutationVariables := map[string]interface{}{
		"input": input,
	}

	response, err := s.executeMutation(mutation, mutationVariables)
	if err != nil {
		return "", err
	}

	return response[mutationName].(map[string]interface{})["id"].(string), nil
}

func (s *Manager) executeMutation(query string, variables map[string]interface{}) (map[string]interface{}, error) {
	req := graphql.NewRequest(query)
	for key, value := range variables {
		req.Var(key, value)
	}
	var response map[string]interface{}
	if err := s.Client.Run(context.Background(), req, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func (s *Manager) executeQuery(query string, variables map[string]interface{}, responseKey string) (map[string]interface{}, error) {
	req := graphql.NewRequest(query)
	for key, value := range variables {
		req.Var(key, value)
	}
	var response map[string]interface{}
	if err := s.Client.Run(context.Background(), req, &response); err != nil {
		return nil, err
	}
	responseData, ok := response[responseKey].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to extract data from response")
	}
	return responseData, nil
}
