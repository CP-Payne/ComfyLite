# 🧩 Custom Workflow Integration Guide

This guide will show you how to add your own workflows to ComfyLite using ComfyUI exports, and how to wire them up with configuration files.

---

## 1. Exporting Your Workflow from ComfyUI

1. Open your ComfyUI instance.
2. Build or modify your desired workflow.
3. Click **"Export (API)"** in ComfyUI.
4. Save the file with a descriptive name (e.g., `my_custom_workflow.json`).

---

## 2. Adding the Workflow to ComfyLite

1. Move your exported `.json` file to the `templates/` directory.


## 3. Creating a Config Mapping File
Each workflow requires a mapping config in YAML that maps request fields to nodes and inputs.

Create a config file under `configs/`. 
>Note: Make sure the config file name matches the exported workflow file.

### Example YAML Config:

`configs/my_custom_workflow.yaml`
```yaml
node_mappings:
  prompt:
    node_id: "6" 
    property: "text"
  seed:
    node_id: "3"
    property: "seed"
  width:
    node_id: "5"
    property: "width"
  height:
    node_id: "5"
    property: "height"
  imageCount:
    node_id: "5"
    property: "batch_size"
```
The keys (`prompt`, `width`, etc.) corresponds to fields in the API request body.

The values map to the keys in the workflow file. For example, the key `prompt` contains `node_id` which defines the node in the workflow file to modify, the property, defines the key within `input` to modify. Internal code example: `params['prompt'] = "some prompt here"` would look in the config file for the key `promp`, it would then go into the workflow.json file and find the object with key: `"6"`, it would then go into the `input` field and find the key `text`, which it would then change the value to `"some prompt here"`. Another example is `imageCount` which maps to the property `batch_size` within the input field of `node 5`. 

## 4. (Optional) Updating the Code for New Parameters
If your workflow uses a new field (e.g, `style_strength`, `negative_prompt` or `weight`):
1. Add the field to the `GenerationRequest` struct in `internal/api/types.go`. 

    ```go
    type GenerationRequest struct {
        Prompt     string `json:"prompt"`
        ImageCount int    `json:"image_count"`
        Width      int    `json:"width"`
        Height     int    `json:"height"`
        WebhookURL string `json:"webhook_url"`
        NegativePrompt string `json:"negative_prompt"` // <-- new field
    }
    ```
2. Update the HTTP handler to pass the field to the service layer if necessary.
    ```go
    // Make sure to add your own validation if necessary.
    promptParams["negativePrompt"] = generationRequest.NegativePrompt
    ```
    `promptsParams` is a map that is pass to the `service` which uses the `workflow` manager to modify the specified workflow with the provided inputs in the map.


    
3. Map the new field in your YAML config.
    ```yaml
    node_mappings:
        # ...
        negativePrompt: #--> This key must match the key in `promptParams` map from step 2.
            node_id: "6" # --> The ID of the ComfyUI node to modify.
            property: "negative_prompt" # --> The specific property key within the node's inputs object.
    ```

4. Specify the new workflow.

    Open `internal/api/handler.go` and change the global constant variable `workflowName` to the new workflow.
    ```go
    const workflowName = "my_custom_workflow"
    ```
    Note: This is for setting the default workflow, and future planned features will allow clients to specify a workflow per request. 

## ✅ That's it!

Once your template and config are in place, you can:
```bash
curl -X POST http://localhost:8083/generate \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "a cat on a rocket",
    ...
  }'
```
