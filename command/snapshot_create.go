package command

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/rancher/go-rancher/client"
)

var (
	MaximumVolumeNameSize = 64
	parsePattern          = regexp.MustCompile(`(.*):(\d+)`)
)

type SnapshotCreateCommand struct {
	Meta
	Name   string
	Labels map[string]string
}
type SnapshotInput struct {
	client.Resource
	Name        string            `json:"name"`
	UserCreated bool              `json:"usercreated"`
	Created     string            `json:"created"`
	Labels      map[string]string `json:"labels"`
}

type SnapshotOutput struct {
	client.Resource
}

// StringSlice is an opaque type for []string to satisfy flag.Value
type StringSlice []string

// Set appends the string value to the list of values
func (f *StringSlice) Set(value string) error {
	*f = append(*f, value)
	return nil
}

// String returns a readable representation of this value (for usage defaults)
func (f *StringSlice) String() string {
	return fmt.Sprintf("%s", *f)
}

// Value returns the slice of strings set by this flag
func (f *StringSlice) Value() []string {
	return *f
}

// Help shows helpText for a particular CLI command
func (c *SnapshotCreateCommand) Help() string {
	helpText := `
    Usage: maya vsm-snapshot <vsm-name> 

  This command will create the snapshot of a given Vsm.

`
	return strings.TrimSpace(helpText)
}

// Synopsis shows short information related to CLI command
func (c *SnapshotCreateCommand) Synopsis() string {
	return "Create snapshot of a VSM"
}

// Run holds the flag values for CLI subcommands
func (c *SnapshotCreateCommand) Run(args []string) int {
	var (
		labelMap map[string]string
		err      error
	)

	flags := c.Meta.FlagSet("vsm-snapshot", FlagSetClient)
	flags.Usage = func() { c.Ui.Output(c.Help()) }

	flags.StringVar(&c.Name, "name", "", "")
	//flags.String(&c.Labels, "label", "")

	if err := flags.Parse(args); err != nil {
		return 1
	}
	/* var name string
	   if len(c.Args()) > 0 {
	       name = c.Args()[0]
	   } */
	/*	var flagset *flag.FlagSet
		labels := lookupStringSlice("label", flagset)
		fmt.Sprint(labels)
		if labels != nil {
			labelMap, err = ParseLabels(labels)
			if err != nil {
				fmt.Printf("cannot parse backup labels")
				return 1
			}
		}
	*/
	//	str := os.Args[1:]
	//	labelMap = map[str]string
	fmt.Sprint(labelMap)
	var client ControllerClient
	id, err := client.Snapshot(c.Name, labelMap)
	if err != nil {
		log.Fatalf("Error running create snapshot command: %v", err)
		return 1
	}

	fmt.Println(id)
	return 0

}

func (c *ControllerClient) Snapshot(name string, labels map[string]string) (string, error) {
	err := GetVsm(nil)
	if err != nil {
		return "", err
	}

	input := SnapshotInput{
		Name:   name,
		Labels: labels,
	}
	output := SnapshotOutput{}
	err = c.post("/v1/volumes/?action=snapshot", input, &output)
	if err != nil {
		return "", err
	}

	return output.Id, err
}

func (c *ControllerClient) post(path string, req, resp interface{}) error {
	return c.do("POST", path, req, resp)
}

func (c *ControllerClient) do(method, path string, req, resp interface{}) error {
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	bodyType := "application/json"
	url := path
	if !strings.HasPrefix(url, "http") {
		url = c.address + path

	}
	log.Printf("%s %s", method, url)
	//fmt.Printf("%s %s", method, url)
	httpReq, err := http.NewRequest(method, url, bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", bodyType)

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 300 {
		content, _ := ioutil.ReadAll(httpResp.Body)
		return fmt.Errorf("Bad response: %d %s: %s", httpResp.StatusCode, httpResp.Status, content)
	}

	if resp == nil {
		return nil
	}
	return json.NewDecoder(httpResp.Body).Decode(resp)
}

/*func (r *Remote) Snapshot(name string, userCreated bool, created string, labels map[string]string) error {
	fmt.Println("Snapshot: %s %s UserCreated %v Created at %v, Labels %v",
		r.name, name, userCreated, created, labels)
	return r.doAction("snapshot",
		&map[string]interface{}{
			"name":        name,
			"usercreated": userCreated,
			"created":     created,
			"labels":      labels,
		})
}*/

func ParseLabels(labels []string) (map[string]string, error) {
	result := map[string]string{}
	for _, label := range labels {
		kv := strings.Split(label, "=")
		if len(kv) != 2 {
			return nil, fmt.Errorf("Invalid label not in <key>=<value> format %v", label)
		}
		key := kv[0]
		value := kv[1]
		//Well, we should rename that ValidVolumeName
		if !ValidVolumeName(key) {
			return nil, fmt.Errorf("Invalid key %v for label %v", key, label)
		}
		if !ValidVolumeName(value) {
			return nil, fmt.Errorf("Invalid value %v for label %v", value, label)
		}
		result[key] = value
	}
	return result, nil
}

func ValidVolumeName(name string) bool {
	if len(name) > MaximumVolumeNameSize {
		return false
	}
	validName := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]+$`)
	return validName.MatchString(name)
}

func lookupStringSlice(name string, set *flag.FlagSet) []string {
	f := set.Lookup(name)
	if f != nil {
		return (f.Value.(*StringSlice)).Value()

	}

	return nil
}
