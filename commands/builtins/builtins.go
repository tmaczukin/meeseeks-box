package builtins

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gomeeseeks/meeseeks-box/jobs/logs"

	"github.com/gomeeseeks/meeseeks-box/jobs"
	"github.com/gomeeseeks/meeseeks-box/tokens"

	"github.com/gomeeseeks/meeseeks-box/auth"
	"github.com/gomeeseeks/meeseeks-box/command"
	"github.com/gomeeseeks/meeseeks-box/template"
	"github.com/gomeeseeks/meeseeks-box/version"
	"github.com/renstrom/dedent"
)

// Builtin Commands Names
const (
	BuiltinVersionCommand   = "version"
	BuiltinHelpCommand      = "help"
	BuiltinGroupsCommand    = "groups"
	BuiltinJobsCommand      = "jobs"
	BuiltinFindJobCommand   = "job"
	BuiltinAuditCommand     = "audit"
	BuiltinAuditJobCommand  = "auditjob"
	BuiltinAuditLogsCommand = "auditlogs"
	BuiltinLastCommand      = "last"
	BuiltinTailCommand      = "tail"
	BuiltinLogsCommand      = "logs"
	BuiltinCancelJobCommand = "cancel"
	BuiltinKillJobCommand   = "kill"

	BuiltinNewAPITokenCommand    = "token-new"
	BuiltinListAPITokenCommand   = "tokens"
	BuiltinRevokeAPITokenCommand = "token-revoke"
)

// Commands is the basic set of builtin commands
var Commands = map[string]command.Command{
	// The help builtin command needs a pointer to the map of generated commands,
	// because of this it is added as the last one when building the whole command
	// map
	BuiltinVersionCommand: versionCommand{
		help: help{"prints the running meeseeks version"},
		cmd:  cmd{BuiltinVersionCommand},
	},
	BuiltinGroupsCommand: groupsCommand{
		help: help{"prints the configured groups"},
		cmd:  cmd{BuiltinGroupsCommand},
	},
	BuiltinJobsCommand: jobsCommand{
		help: help{"shows the last executed jobs for the calling user, accepts -limit"},
		cmd:  cmd{BuiltinJobsCommand},
	},
	BuiltinAuditCommand: auditCommand{
		help: help{"lists jobs from all users or a specific one (admin only), accepts -user and -limit to filter."},
		cmd:  cmd{BuiltinAuditCommand},
	},
	BuiltinAuditJobCommand: auditJobCommand{
		help: help{"shows a command metadata by job ID from any user (admin only)"},
		cmd:  cmd{BuiltinAuditJobCommand},
	},
	BuiltinAuditLogsCommand: auditLogsCommand{
		help: help{"shows the logs of any command by job ID (admin only)"},
		cmd:  cmd{BuiltinAuditLogsCommand},
	},
	BuiltinLastCommand: lastCommand{
		help: help{"shows the last executed command by the calling user"},
		cmd:  cmd{BuiltinLastCommand},
	},
	BuiltinFindJobCommand: findJobCommand{
		help: help{"find one job by id"},
		cmd:  cmd{BuiltinFindJobCommand},
	},
	BuiltinTailCommand: tailCommand{
		help: help{"returns the last command output or error"},
		cmd:  cmd{BuiltinTailCommand},
	},
	BuiltinLogsCommand: logsCommand{
		help: help{"returns the logs of the command id passed as argument"},
		cmd:  cmd{BuiltinLogsCommand},
	},
	BuiltinNewAPITokenCommand: newAPITokenCommand{
		help: help{"creates a new API token for the calling user, channel and command with args, requires at least #channel and command"},
		cmd:  cmd{BuiltinNewAPITokenCommand},
	},
	BuiltinListAPITokenCommand: listAPITokensCommand{
		help: help{"lists the API tokens"},
		cmd:  cmd{BuiltinListAPITokenCommand},
	},
	BuiltinRevokeAPITokenCommand: revokeAPITokenCommand{
		help: help{"revokes an API token"},
		cmd:  cmd{BuiltinRevokeAPITokenCommand},
	},
}

// AddHelpCommand creates a new help command and adds it to the map
func AddHelpCommand(c map[string]command.Command) {
	c[BuiltinHelpCommand] = helpCommand{
		commands: c,
		cmd:      cmd{BuiltinHelpCommand},
		help:     help{"prints all the kwnown commands and its associated help"},
	}
}

type plainTemplates struct{}

func (p plainTemplates) Templates() map[string]string {
	return map[string]string{
		template.SuccessKey: fmt.Sprintf("{{ .user }} {{ AnyValue \"%s\" . }}{{ with $out := .output }}\n{{ $out }}{{ end }}", template.SuccessKey),
	}
}

type defaultTemplates struct {
}

func (d defaultTemplates) Templates() map[string]string {
	return template.GetDefaultTemplates()
}

type defaultTimeout struct{}

func (d defaultTimeout) Timeout() time.Duration {
	return command.DefaultCommandTimeout
}

type emptyArgs struct{}

func (b emptyArgs) Args() []string {
	return []string{}
}

type noRecord struct{}

func (n noRecord) Record() bool {
	return false
}

type allowAll struct{}

func (a allowAll) AuthStrategy() string {
	return auth.AuthStrategyAny
}

func (a allowAll) AllowedGroups() []string {
	return []string{}
}

type allowAdmins struct{}

func (a allowAdmins) AuthStrategy() string {
	return auth.AuthStrategyAllowedGroup
}

func (a allowAdmins) AllowedGroups() []string {
	return []string{auth.AdminGroup}
}

type noHandshake struct {
}

func (b noHandshake) HasHandshake() bool {
	return false
}

type help struct {
	help string
}

func (h help) Help() string {
	return h.help
}

type cmd struct {
	cmd string
}

func (c cmd) Cmd() string {
	return c.cmd
}

type versionCommand struct {
	cmd
	help
	noHandshake
	noRecord
	allowAll
	plainTemplates
	emptyArgs
	defaultTimeout
}

func (v versionCommand) Execute(_ context.Context, job jobs.Job) (string, error) {
	return fmt.Sprintf("meeseeks-box version %s, commit %s, built at %s",
		version.Version, version.Commit, version.Date), nil
}

type helpCommand struct {
	cmd
	help
	noHandshake
	noRecord
	allowAll
	plainTemplates
	emptyArgs
	defaultTimeout
	commands map[string]command.Command
}

var helpTemplate = dedent.Dedent(`
	{{ range $name, $cmd := .commands }}- {{ $name }}: {{ $cmd.Help }}
	{{ end }}`)

func (h helpCommand) Execute(_ context.Context, job jobs.Job) (string, error) {
	tmpl, err := template.New("help", helpTemplate)
	if err != nil {
		return "", err
	}
	return tmpl.Render(template.Payload{
		"commands": h.commands,
	})
}

type cancelJobCommand struct {
	cmd
	help
	noHandshake
	noRecord
	emptyArgs
	allowAll
	defaultTemplates
	defaultTimeout
	cancelFunc func(jobID uint64)
}

// NewCancelJobCommand creates a command that will invoke the passed cancel job function when executed
func NewCancelJobCommand(f func(jobID uint64)) command.Command {
	return cancelJobCommand{
		help:       help{"cancels a jobs owned by the calling user that is currently running"},
		cancelFunc: f,
	}
}

func (c cancelJobCommand) Execute(_ context.Context, job jobs.Job) (string, error) {
	jobID, err := parseJobID(job)
	if err != nil {
		return "", err
	}
	j, err := jobs.Get(jobID)
	if err != nil {
		return "", err
	}
	if job.Request.Username != j.Request.Username {
		return "", jobs.ErrNoJobWithID
	}
	c.cancelFunc(jobID)
	return fmt.Sprintf("Issued command cancellation to job %d", jobID), nil
}

type killJobCommand struct {
	cmd
	help
	noHandshake
	noRecord
	emptyArgs
	allowAdmins
	defaultTemplates
	defaultTimeout
	cancelFunc func(jobID uint64)
}

// NewKillJobCommand creates a command that will invoke the passed cancel job function when executed
func NewKillJobCommand(f func(jobID uint64)) command.Command {
	return killJobCommand{
		help:       help{"cancels a jobs that is currently running, from any user"},
		cancelFunc: f,
	}
}

func (k killJobCommand) Execute(_ context.Context, job jobs.Job) (string, error) {
	jobID, err := parseJobID(job)
	if err != nil {
		return "", err
	}
	_, err = jobs.Get(jobID)
	if err != nil {
		return "", err
	}
	k.cancelFunc(jobID)
	return fmt.Sprintf("Issued command cancellation to job %d", jobID), nil
}

type groupsCommand struct {
	cmd
	help
	noHandshake
	noRecord
	emptyArgs
	allowAdmins
	plainTemplates
	defaultTimeout
}

var groupsTemplate = dedent.Dedent(`
	{{- range $group, $users := .groups }}
	- {{ $group }}:
		{{- range $index, $user := $users }}{{ if ne $index 0 }},{{ end }} {{ $user }}{{ end }}
	{{- end }}
	`)

func (g groupsCommand) Execute(_ context.Context, job jobs.Job) (string, error) {
	tmpl, err := template.New("version", groupsTemplate)
	if err != nil {
		return "", err
	}
	return tmpl.Render(template.Payload{
		"groups": auth.GetGroups(),
	})
}

type jobsCommand struct {
	cmd
	help
	noHandshake
	noRecord
	allowAll
	plainTemplates
	emptyArgs
	defaultTimeout
}

var jobsTemplate = strings.Join([]string{
	"{{- $length := len .jobs }}{{- if eq $length 0 }}",
	"No jobs found\n",
	"{{ else }}",
	"{{- range $job := .jobs }}",
	"*{{ $job.ID }}* - {{ HumanizeTime $job.StartTime }}",
	" - *{{ $job.Request.Command }}*",
	" by *{{ $job.Request.Username }}*",
	" in *{{ if $job.Request.IsIM }}DM{{ else }}{{ $job.Request.ChannelLink }}{{ end }}*",
	" - *{{ $job.Status }}*\n",
	"{{ end }}",
	"{{ end }}",
}, "")

func (j jobsCommand) Execute(_ context.Context, job jobs.Job) (string, error) {
	flags := flag.NewFlagSet("jobs", flag.ContinueOnError)
	limit := flags.Int("limit", 5, "how many jobs to return")
	status := flags.String("status", "", "filter jobs per status (running, failed or successful)")
	if err := flags.Parse(job.Request.Args); err != nil {
		return "", err
	}

	callingUser := job.Request.Username
	requestedStatus := strings.Title(*status)
	jobs, err := jobs.Find(jobs.JobFilter{
		Limit: *limit,
		Match: jobs.MultiMatch(
			isUser(callingUser),
			isStatusOrEmpty(requestedStatus),
		),
	})

	if err != nil {
		return "", err
	}
	tmpl, err := template.New("jobs", jobsTemplate)
	if err != nil {
		return "", err
	}
	return tmpl.Render(template.Payload{
		"jobs": jobs,
	})
}

type auditCommand struct {
	cmd
	help
	noHandshake
	noRecord
	allowAdmins
	plainTemplates
	emptyArgs
	defaultTimeout
}

func (j auditCommand) Execute(_ context.Context, job jobs.Job) (string, error) {
	flags := flag.NewFlagSet("audit", flag.ContinueOnError)
	limit := flags.Int("limit", 5, "how many jobs to return")
	user := flags.String("user", "", "the user to audit")
	status := flags.String("status", "", "filter jobs per status (running, failed or successful)")
	if err := flags.Parse(job.Request.Args); err != nil {
		return "", err
	}

	requestedStatus := strings.Title(*status)

	jobs, err := jobs.Find(jobs.JobFilter{
		Limit: *limit,
		Match: jobs.MultiMatch(
			isStatusOrEmpty(requestedStatus),
			func(j jobs.Job) bool {
				if *user == "" {
					return true
				}
				return *user == j.Request.Username
			},
		),
	})

	if err != nil {
		return "", err
	}
	tmpl, err := template.New("jobs", jobsTemplate)
	if err != nil {
		return "", err
	}
	return tmpl.Render(template.Payload{
		"jobs": jobs,
	})
}

type lastCommand struct {
	cmd
	help
	noHandshake
	noRecord
	allowAll
	plainTemplates
	emptyArgs
	defaultTimeout
}

var jobTemplate = `
{{- with $job := .job }}{{ with $r := $job.Request }}* *ID* {{ $job.ID }}
* *Status* {{ $job.Status}}
* *Command* {{ $r.Command }}{{ with $args := $r.Args }}
* *Args* "{{ Join $args "\" \"" }}" {{ end }}
* *Where* {{ if $r.IsIM }}IM{{ else }}{{ $r.ChannelLink }}{{ end }}
* *When* {{ HumanizeTime $job.StartTime }}
{{- end }}{{- end }}
`

func (l lastCommand) Execute(_ context.Context, job jobs.Job) (string, error) {
	callingUser := job.Request.Username
	jobs, err := jobs.Find(jobs.JobFilter{
		Limit: 1,
		Match: isUser(callingUser),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get the last job: %s", err)
	}
	if len(jobs) == 0 {
		return "", fmt.Errorf("No last command for current user")
	}
	tmpl, err := template.New("job", jobTemplate)
	if err != nil {
		return "", err
	}
	return tmpl.Render(template.Payload{
		"job": jobs[0],
	})
}

type findJobCommand struct {
	cmd
	help
	noHandshake
	noRecord
	allowAll
	plainTemplates
	emptyArgs
	defaultTimeout
}

func (l findJobCommand) Execute(_ context.Context, job jobs.Job) (string, error) {
	id, err := parseJobID(job)
	if err != nil {
		return "", err
	}

	callingUser := job.Request.Username
	jobs, err := jobs.Find(jobs.JobFilter{
		Limit: 1,
		Match: jobs.MultiMatch(
			isUser(callingUser),
			isJobID(id)),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get the last job: %s", err)
	}
	if len(jobs) == 0 {
		return "", fmt.Errorf("No last command for current user")
	}

	tmpl, err := template.New("job", jobTemplate)
	if err != nil {
		return "", err
	}
	return tmpl.Render(template.Payload{
		"job": jobs[0],
	})
}

type auditJobCommand struct {
	cmd
	help
	noHandshake
	noRecord
	allowAdmins
	plainTemplates
	emptyArgs
	defaultTimeout
}

func (l auditJobCommand) Execute(_ context.Context, job jobs.Job) (string, error) {
	id, err := parseJobID(job)
	if err != nil {
		return "", err
	}

	jobs, err := jobs.Find(jobs.JobFilter{
		Limit: 1,
		Match: isJobID(id),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get job %d: %s", job.ID, err)
	}
	if len(jobs) == 0 {
		return "", fmt.Errorf("Job not found")
	}

	tmpl, err := template.New("job", jobTemplate)
	if err != nil {
		return "", err
	}
	return tmpl.Render(template.Payload{
		"job": jobs[0],
	})
}

type auditLogsCommand struct {
	cmd
	help
	noHandshake
	noRecord
	allowAll
	defaultTemplates
	emptyArgs
	defaultTimeout
}

func (t auditLogsCommand) Execute(_ context.Context, job jobs.Job) (string, error) {
	id, err := parseJobID(job)
	if err != nil {
		return "", err
	}

	jobs, err := jobs.Find(jobs.JobFilter{
		Limit: 1,
		Match: isJobID(id),
	})

	if err != nil {
		return "", fmt.Errorf("failed to find job with id %d: %s", id, err)
	}
	if len(jobs) == 0 {
		return "", fmt.Errorf("there is no job %d", id)
	}
	j := jobs[0]

	jobLogs, err := logs.Get(j.ID)
	if err != nil {
		return "", err
	}
	return jobLogs.Output, jobLogs.GetError()
}

type tailCommand struct {
	cmd
	help
	noHandshake
	noRecord
	allowAll
	defaultTemplates
	emptyArgs
	defaultTimeout
}

func (t tailCommand) Execute(_ context.Context, job jobs.Job) (string, error) {
	callingUser := job.Request.Username
	jobs, err := jobs.Find(jobs.JobFilter{
		Limit: 1,
		Match: isUser(callingUser),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get the last job: %s", err)
	}
	if len(jobs) == 0 {
		return "", fmt.Errorf("No last command for current user")
	}
	j := jobs[0]

	jobLogs, err := logs.Get(j.ID)
	if err != nil {
		return "", err
	}
	return jobLogs.Output, jobLogs.GetError()
}

type logsCommand struct {
	cmd
	help
	noHandshake
	noRecord
	allowAll
	defaultTemplates
	emptyArgs
	defaultTimeout
}

func (t logsCommand) Execute(_ context.Context, job jobs.Job) (string, error) {
	id, err := parseJobID(job)
	if err != nil {
		return "", err
	}

	callingUser := job.Request.Username
	jobs, err := jobs.Find(jobs.JobFilter{
		Limit: 1,
		Match: jobs.MultiMatch(
			isUser(callingUser),
			isJobID(id)),
	})
	if err != nil {
		return "", fmt.Errorf("failed to find job with id %d: %s", id, err)
	}
	if len(jobs) == 0 {
		return "", fmt.Errorf("No job with id %d for user %s", id, callingUser)
	}
	j := jobs[0]

	jobLogs, err := logs.Get(j.ID)
	if err != nil {
		return "", err
	}
	return jobLogs.Output, jobLogs.GetError()
}

type newAPITokenCommand struct {
	cmd
	help
	noHandshake
	noRecord
	allowAdmins
	plainTemplates
	emptyArgs
	defaultTimeout
}

func (n newAPITokenCommand) Execute(_ context.Context, job jobs.Job) (string, error) {
	if !job.Request.IsIM {
		return "", fmt.Errorf("API tokens can only be managed over an IM conversation, security ffs")
	}
	if len(job.Request.Args) < 3 {
		return "", fmt.Errorf("not enough arguments passed in")
	}

	t, err := tokens.Create(tokens.NewTokenRequest{
		UserLink:    job.Request.Args[0],
		ChannelLink: job.Request.Args[1],
		Text:        strings.Join(job.Request.Args[2:], " "),
	})
	return fmt.Sprintf("created token %s", t), err
}

type revokeAPITokenCommand struct {
	cmd
	help
	noHandshake
	noRecord
	allowAdmins
	plainTemplates
	emptyArgs
	defaultTimeout
}

func (r revokeAPITokenCommand) Execute(_ context.Context, job jobs.Job) (string, error) {
	if !job.Request.IsIM {
		return "", fmt.Errorf("API tokens can only be managed over an IM conversation, security ffs")
	}
	if len(job.Request.Args) != 1 {
		return "", fmt.Errorf("only one token ID should be passed as an argument")
	}
	tokenID := job.Request.Args[0]
	if err := tokens.Revoke(tokenID); err != nil {
		return "", err
	}
	return fmt.Sprintf("Token *%s* has been revoked", tokenID), nil
}

type listAPITokensCommand struct {
	cmd
	help
	noHandshake
	noRecord
	allowAdmins
	plainTemplates
	emptyArgs
	defaultTimeout
}

var listTokensTemplate = `{{ if eq (len .tokens) 0 }}No tokens could be found{{ else }}{{ range $t := .tokens }}- *{{ $t.TokenID }}* {{ $t.UserLink }} at {{ $t.ChannelLink }} _{{ $t.Text}}_
{{ end }}{{ end }}`

func (l listAPITokensCommand) Execute(_ context.Context, job jobs.Job) (string, error) {
	if !job.Request.IsIM {
		return "", fmt.Errorf("API tokens can only be managed over an IM conversation, security ffs")
	}

	flags := flag.NewFlagSet("jobs", flag.ContinueOnError)
	limit := flags.Int("limit", 5, "how many jobs to return")
	user := flags.String("user", "", "user to filter for")
	channel := flags.String("channel", "", "channel to filter for")
	command := flags.String("command", "", "command to filter for")

	flags.Parse(job.Request.Args)

	tmpl, err := template.New("tokens", listTokensTemplate)
	if err != nil {
		return "", err
	}

	matchers := []func(tokens.Token) bool{}
	if *user != "" {
		matchers = append(matchers, func(t tokens.Token) bool {
			return t.UserLink == *user
		})
	}
	if *channel != "" {
		matchers = append(matchers, func(t tokens.Token) bool {
			return t.ChannelLink == *channel
		})
	}
	if *command != "" {
		matchers = append(matchers, func(t tokens.Token) bool {
			return strings.HasPrefix(t.Text, *command)
		})
	}

	t, err := tokens.Find(tokens.Filter{
		Limit: *limit,
		Match: tokens.MultiMatch(matchers...),
	})
	if err != nil {
		return "", err
	}

	return tmpl.Render(template.Payload{
		"tokens": t,
	})
}

func parseJobID(job jobs.Job) (uint64, error) {
	if len(job.Request.Args) == 0 {
		return 0, fmt.Errorf("no job id passed")
	}
	id, err := strconv.ParseUint(job.Request.Args[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid job ID %s: %s", job.Request.Args[0], err)
	}

	return id, nil
}

func isUser(username string) func(jobs.Job) bool {
	return func(j jobs.Job) bool {
		return j.Request.Username == username
	}
}

func isJobID(jobID uint64) func(jobs.Job) bool {
	return func(j jobs.Job) bool {
		return j.ID == jobID
	}
}

func isStatusOrEmpty(status string) func(jobs.Job) bool {
	return func(j jobs.Job) bool {
		if status == "" {
			return true
		}
		return j.Status == status
	}
}
