package google

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/cloudfunctions/v1beta2"
	"google.golang.org/api/googleapi"
)

func resourceCloudFunction() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudFunctionCreate,
		Read:   resourceCloudFunctionRead,
		Update: resourceCloudFunctionUpdate,
		Delete: resourceCloudFunctionDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"available_memory_mb": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"entry_point": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"event_trigger": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"event_type": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"resource": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"https_trigger": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"url": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"location": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"service_account": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"source_archive_url": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"source_repository": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"branch": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"deployed_revision": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"repository_url": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"revision": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"source_path": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"tag": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"timeout": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceCloudFunctionCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	rawLocation := config.Region
	if v, ok := d.GetOk("location"); ok {
		rawLocation = v.(string)
	}
	location := fmt.Sprintf("projects/%s/locations/%s", project, rawLocation)

	name := d.Get("name").(string)
	id := fmt.Sprintf("projects/%s/locations/%s/functions/%s", project, rawLocation, name)

	function := &cloudfunctions.CloudFunction{
		Name: id,
	}

	if v, ok := d.GetOk("available_memory_mb"); ok {
		function.AvailableMemoryMb = int64(v.(int))
	}
	if v, ok := d.GetOk("entry_point"); ok {
		function.EntryPoint = v.(string)
	}
	if v, ok := d.GetOk("source_archive_url"); ok {
		function.SourceArchiveUrl = v.(string)
	}
	if v, ok := d.GetOk("timeout"); ok {
		function.Timeout = v.(string)
	}
	if v, ok := d.GetOk("event_trigger"); ok {
		function.EventTrigger = expandFunctionEventTrigger(v.([]interface{}))
	}
	if v, ok := d.GetOk("https_trigger"); ok {
		function.HttpsTrigger = expandFunctionHttpsTrigger(v.([]interface{}))
	}
	if v, ok := d.GetOk("source_repository"); ok {
		function.SourceRepository = expandFunctionSourceRepository(v.([]interface{}))
	}

	log.Printf("[INFO] Creating new Cloud Function: %#v", function)
	call := config.clientFunctions.Projects.Locations.Functions.Create(location, function)
	_, err = call.Do()
	if err != nil {
		return err
	}

	d.SetId(function.Name)

	stateConf := resource.StateChangeConf{
		Target: []string{"READY", "FAILED"},
		Pending: []string{
			"STATUS_UNSPECIFIED",
			"DEPLOYING",
		},
		Timeout: 5 * time.Minute,
		Refresh: func() (interface{}, string, error) {
			call := config.clientFunctions.Projects.Locations.Functions.Get(function.Name)
			resp, err := call.Do()
			if err != nil {
				return 42, "", err
			}

			if resp.Status == "FAILED" {
				call := config.clientFunctions.Operations.Get(resp.LatestOperation)
				out, err := call.Do()
				if err != nil {
					return 42, "", err
				}
				reason := ""
				if out.Error != nil {
					reason = out.Error.Message
				}

				return 42, "FAILED", fmt.Errorf("Deploying cloud function %q failed: %q", name, reason)
			}

			return resp, resp.Status, nil
		},
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resourceCloudFunctionRead(d, meta)
}

func resourceCloudFunctionRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	name := d.Id()
	call := config.clientFunctions.Projects.Locations.Functions.Get(name)
	resp, err := call.Do()
	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("Cloud Function %q", name))
	}

	if resp.AvailableMemoryMb != 0 {
		d.Set("available_memory_mb", resp.AvailableMemoryMb)
	}
	if resp.EntryPoint != "" {
		d.Set("entry_point", resp.EntryPoint)
	}
	if resp.SourceArchiveUrl != "" {
		d.Set("source_archive_url", resp.SourceArchiveUrl)
	}
	if resp.Timeout != "" {
		d.Set("timeout", resp.Timeout)
	}
	if resp.EventTrigger != nil {
		d.Set("event_trigger", flattenFunctionEventTrigger(resp.EventTrigger))
	}
	if resp.HttpsTrigger != nil {
		d.Set("https_trigger", flattenFunctionHttpsTrigger(resp.HttpsTrigger))
	}
	if resp.SourceRepository != nil {
		d.Set("source_repository", flattenFunctionSourceRepository(resp.SourceRepository))
	}

	return nil
}

func resourceCloudFunctionUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	name := d.Id()

	// TODO: is this patch behaviour or do we send all the arguments?
	function := &cloudfunctions.CloudFunction{}

	call := config.clientFunctions.Projects.Locations.Functions.Update(name, function)
	_, err := call.Do()
	if err != nil {
		return err
	}

	return nil
}

func resourceCloudFunctionDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	name := d.Id()
	call := config.clientFunctions.Projects.Locations.Functions.Delete(name)
	_, err := call.Do()
	if err != nil {
		return err
	}

	stateConf := resource.StateChangeConf{
		Target: []string{},
		Pending: []string{
			"STATUS_UNSPECIFIED",
			"DEPLOYING",
			"DELETING",
		},
		Timeout: 5 * time.Minute,
		Refresh: func() (interface{}, string, error) {
			call := config.clientFunctions.Projects.Locations.Functions.Get(name)
			resp, err := call.Do()
			if err != nil {
				if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
					return nil, "", nil
				}
				return 42, "", err
			}

			if resp.Status == "FAILED" {
				return 42, "", fmt.Errorf("Deletion of function %q failed.", name)
				// TODO: Is there any way to find cause of the error and display it to the user?
			}

			return resp, resp.Status, err
		},
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return nil
}

func expandFunctionEventTrigger(l []interface{}) *cloudfunctions.EventTrigger {
	if len(l) == 0 || l[0] == nil {
		return nil
	}
	in := l[0].(map[string]interface{})
	obj := cloudfunctions.EventTrigger{}

	if v, ok := in["event_type"].(string); ok {
		obj.EventType = v
	}

	if v, ok := in["resource"].(string); ok {
		obj.Resource = v
	}

	return &obj
}

func expandFunctionHttpsTrigger(l []interface{}) *cloudfunctions.HTTPSTrigger {
	if len(l) == 0 || l[0] == nil {
		return nil
	}
	in := l[0].(map[string]interface{})
	obj := cloudfunctions.HTTPSTrigger{}

	if v, ok := in["url"].(string); ok {
		obj.Url = v
	}

	return &obj
}

func expandFunctionSourceRepository(l []interface{}) *cloudfunctions.SourceRepository {
	if len(l) == 0 || l[0] == nil {
		return nil
	}
	in := l[0].(map[string]interface{})
	obj := cloudfunctions.SourceRepository{}

	if v, ok := in["branch"].(string); ok {
		obj.Branch = v
	}
	if v, ok := in["deployed_revision"].(string); ok {
		obj.DeployedRevision = v
	}
	if v, ok := in["repository_url"].(string); ok {
		obj.RepositoryUrl = v
	}
	if v, ok := in["revision"].(string); ok {
		obj.Revision = v
	}
	if v, ok := in["source_path"].(string); ok {
		obj.SourcePath = v
	}
	if v, ok := in["tag"].(string); ok {
		obj.Tag = v
	}

	return &obj
}

func flattenFunctionEventTrigger(in *cloudfunctions.EventTrigger) []interface{} {
	att := make(map[string]interface{})

	if in.EventType != "" {
		att["event_type"] = in.EventType
	}

	if in.Resource != "" {
		att["resource"] = in.Resource
	}

	return []interface{}{att}
}

func flattenFunctionHttpsTrigger(in *cloudfunctions.HTTPSTrigger) []interface{} {
	att := make(map[string]interface{})

	if in.Url != "" {
		att["url"] = in.Url
	}

	return []interface{}{att}
}

func flattenFunctionSourceRepository(in *cloudfunctions.SourceRepository) []interface{} {
	att := make(map[string]interface{})

	if in.Branch != "" {
		att["branch"] = in.Branch
	}
	if in.DeployedRevision != "" {
		att["deployed_revision"] = in.DeployedRevision
	}
	if in.RepositoryUrl != "" {
		att["repository_url"] = in.RepositoryUrl
	}
	if in.Revision != "" {
		att["revision"] = in.Revision
	}
	if in.SourcePath != "" {
		att["source_path"] = in.SourcePath
	}
	if in.Tag != "" {
		att["tag"] = in.Tag
	}

	return []interface{}{att}
}
