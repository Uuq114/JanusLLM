package request

import "encoding/json"

func ExtractModelAndStream(rawBody []byte) (model string, stream bool, err error) {
	if len(rawBody) == 0 {
		return "", false, nil
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rawBody, &body); err != nil {
		return "", false, err
	}

	if modelValue, ok := body["model"]; ok {
		if modelStr, ok := modelValue.(string); ok {
			model = modelStr
		}
	}
	if streamValue, ok := body["stream"]; ok {
		if streamBool, ok := streamValue.(bool); ok {
			stream = streamBool
		}
	}

	return model, stream, nil
}
