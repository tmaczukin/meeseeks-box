package builtins_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/gomeeseeks/meeseeks-box/auth"
	"github.com/gomeeseeks/meeseeks-box/commands"
	"github.com/gomeeseeks/meeseeks-box/commands/builtins"
	"github.com/gomeeseeks/meeseeks-box/jobs"
	"github.com/gomeeseeks/meeseeks-box/jobs/logs"
	"github.com/gomeeseeks/meeseeks-box/meeseeks/request"
	stubs "github.com/gomeeseeks/meeseeks-box/testingstubs"
	"github.com/gomeeseeks/meeseeks-box/tokens"
	"github.com/renstrom/dedent"
)

var basicGroups = map[string][]string{
	"admins": []string{"admin_user"},
	"other":  []string{"user_one", "user_two"},
}

var req = request.Request{
	Command:     "command",
	Channel:     "general",
	ChannelID:   "123",
	ChannelLink: "<#123>",
	UserID:      "someoneID",
	Username:    "someone",
	Args:        []string{"arg1", "arg2"},
}

func Test_BuiltinCommands(t *testing.T) {
	auth.Configure(basicGroups)

	var jobID uint64

	commands.Add(builtins.BuiltinCancelJobCommand, builtins.NewCancelJobCommand(
		func(j uint64) {
			jobID = j
		}))
	commands.Add(builtins.BuiltinKillJobCommand, builtins.NewKillJobCommand(
		func(j uint64) {
			jobID = j
		}))

	tt := []struct {
		name          string
		cmd           string
		job           jobs.Job
		setup         func()
		expected      string
		expectedMatch string
		expectedError error
	}{
		{
			name:     "version command",
			cmd:      builtins.BuiltinVersionCommand,
			job:      jobs.Job{},
			expected: "meeseeks-box version , commit , built at ",
		},
		{
			name: "help command",
			cmd:  builtins.BuiltinHelpCommand,
			job:  jobs.Job{},
			expected: dedent.Dedent(`
				- audit: lists jobs from all users or a specific one (admin only), accepts -user and -limit to filter.
				- auditjob: shows a command metadata by job ID from any user (admin only)
				- auditlogs: shows the logs of any command by job ID (admin only)
				- cancel: cancels a jobs owned by the calling user that is currently running
				- groups: prints the configured groups
				- help: prints all the kwnown commands and its associated help
				- job: find one job by id
				- jobs: shows the last executed jobs for the calling user, accepts -limit
				- kill: cancels a jobs that is currently running, from any user
				- last: shows the last executed command by the calling user
				- logs: returns the logs of the command id passed as argument
				- tail: returns the last command output or error
				- token-new: creates a new API token for the calling user, channel and command with args, requires at least #channel and command
				- token-revoke: revokes an API token
				- tokens: lists the API tokens
				- version: prints the running meeseeks version
				`),
		},
		{
			name: "groups command",
			cmd:  builtins.BuiltinGroupsCommand,
			job:  jobs.Job{},
			expected: dedent.Dedent(`
					- admins: admin_user
					- other: user_one, user_two
					`),
		},
		{
			name: "test jobs command",
			cmd:  builtins.BuiltinJobsCommand,
			job: jobs.Job{
				Request: request.Request{Username: "someone"},
			},
			setup: func() {
				j, err := jobs.Create(req)
				stubs.Must(t, "could not create job", err)
				j.Finish(jobs.SuccessStatus)
			},
			expected: "*1* - now - *command* by *someone* in *<#123>* - *Successful*\n",
		},
		{
			name: "test audit command",
			cmd:  builtins.BuiltinAuditCommand,
			job: jobs.Job{
				Request: request.Request{},
			},
			setup: func() {
				j, err := jobs.Create(req)
				stubs.Must(t, "could not create job", err)
				j.Finish(jobs.SuccessStatus)
			},
			expected: "*1* - now - *command* by *someone* in *<#123>* - *Successful*\n",
		},
		{
			name: "test jobs command with limit",
			cmd:  builtins.BuiltinJobsCommand,
			job: jobs.Job{
				Request: request.Request{Username: "someone", Args: []string{"-limit=1"}},
			},
			setup: func() {
				jobs.Create(req)
				jobs.Create(req)
			},
			expected: "*2* - now - *command* by *someone* in *<#123>* - *Running*\n",
		},
		{
			name: "test jobs command on IM",
			cmd:  builtins.BuiltinJobsCommand,
			job: jobs.Job{
				Request: request.Request{Username: "someone"},
			},
			setup: func() {
				jobs.Create(request.Request{
					Command:   "command",
					Channel:   "general",
					ChannelID: "123",
					Username:  "someone",
					Args:      []string{"arg1", "arg2"},
					IsIM:      true,
				})
			},
			expected: "*1* - now - *command* by *someone* in *DM* - *Running*\n",
		},
		{
			name: "test last command",
			cmd:  builtins.BuiltinLastCommand,
			job: jobs.Job{
				Request: request.Request{Username: "someone"},
			},
			setup: func() {
				jobs.Create(req)
				jobs.Create(req)
				jobs.Create(req)
			},
			expected: "* *ID* 3\n* *Status* Running\n* *Command* command\n* *Args* \"arg1\" \"arg2\" \n* *Where* <#123>\n* *When* now\n",
		},
		{
			name: "test find command",
			cmd:  builtins.BuiltinFindJobCommand,
			job: jobs.Job{
				Request: request.Request{Username: "someone", Args: []string{"1"}},
			},
			setup: func() {
				jobs.Create(req)
				jobs.Create(req)
			},
			expected: "* *ID* 1\n* *Status* Running\n* *Command* command\n* *Args* \"arg1\" \"arg2\" \n* *Where* <#123>\n* *When* now\n",
		},
		{
			name: "test auditjob command",
			cmd:  builtins.BuiltinAuditJobCommand,
			job: jobs.Job{
				Request: request.Request{Username: "someone", Args: []string{"1"}},
			},
			setup: func() {
				jobs.Create(req)
				jobs.Create(req)
			},
			expected: "* *ID* 1\n* *Status* Running\n* *Command* command\n* *Args* \"arg1\" \"arg2\" \n* *Where* <#123>\n* *When* now\n",
		},
		{
			name: "test tail command",
			cmd:  builtins.BuiltinTailCommand,
			job: jobs.Job{
				Request: request.Request{Username: "someone"},
			},
			setup: func() {
				j, err := jobs.Create(req)
				stubs.Must(t, "create job", err)
				logs.Append(j.ID, "something to say 1")

				j, err = jobs.Create(req)
				stubs.Must(t, "create job", err)
				logs.Append(j.ID, "something to say 2")
			},
			expected: "something to say 2",
		},
		{
			name: "test logs command",
			cmd:  builtins.BuiltinLogsCommand,
			job: jobs.Job{
				Request: request.Request{Username: "someone", Args: []string{"1"}},
			},
			setup: func() {
				j, err := jobs.Create(req)
				stubs.Must(t, "create job", err)
				logs.Append(j.ID, "something to say 1")

				j, err = jobs.Create(req)
				stubs.Must(t, "create job", err)
				logs.Append(j.ID, "something to say 2")
			},
			expected: "something to say 1",
		},
		{
			name: "test auditlogs command",
			cmd:  builtins.BuiltinAuditLogsCommand,
			job: jobs.Job{
				Request: request.Request{Username: "admin_user", Args: []string{"1"}},
			},
			setup: func() {
				j, err := jobs.Create(req)
				stubs.Must(t, "create job", err)
				logs.Append(j.ID, "something to say 1")

				j, err = jobs.Create(req)
				stubs.Must(t, "create job", err)
				logs.Append(j.ID, "something to say 2")
			},
			expected: "something to say 1",
		},
		{
			name: "test token-new command",
			cmd:  builtins.BuiltinNewAPITokenCommand,
			job: jobs.Job{
				Request: request.Request{Username: "admin_user", IsIM: true, Args: []string{"admin_user", "yolo", "rm", "-rf"}},
			},
			expectedMatch: "created token .*",
		},
		{
			name: "test tokens command",
			cmd:  builtins.BuiltinListAPITokenCommand,
			job: jobs.Job{
				Request: request.Request{Username: "admin_user", IsIM: true},
			},
			setup: func() {
				_, err := tokens.Create(tokens.NewTokenRequest{
					ChannelLink: "channelLink",
					UserLink:    "userLink",
					Text:        "something",
				})
				stubs.Must(t, "create token", err)

			},
			expectedMatch: "- \\*.*?\\* userLink at channelLink _something_",
		},
		{
			name: "test kill job command",
			cmd:  builtins.BuiltinKillJobCommand,
			job: jobs.Job{
				Request: request.Request{Username: "someone", Args: []string{"1"}},
			},
			setup: func() {
				_, err := jobs.Create(req)
				stubs.Must(t, "create job", err)

				_, err = jobs.Create(req)
				stubs.Must(t, "create job", err)
			},
			expected: "Issued command cancellation to job 1",
		},
		{
			name: "test cancel job command",
			cmd:  builtins.BuiltinCancelJobCommand,
			job: jobs.Job{
				Request: request.Request{Username: "someone", Args: []string{"2"}},
			},
			setup: func() {
				_, err := jobs.Create(req)
				stubs.Must(t, "create job", err)

				_, err = jobs.Create(req)
				stubs.Must(t, "create job", err)
			},
			expected: "Issued command cancellation to job 2",
		},
		{
			name: "test cancel job command with wrong user",
			cmd:  builtins.BuiltinCancelJobCommand,
			job: jobs.Job{
				Request: request.Request{Username: "someone_else", Args: []string{"2"}},
			},
			setup: func() {
				_, err := jobs.Create(req)
				stubs.Must(t, "create job", err)

				_, err = jobs.Create(req)
				stubs.Must(t, "create job", err)
			},
			expectedError: fmt.Errorf("no job could be found"),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			stubs.Must(t, "failed to run tests", stubs.WithTmpDB(func(_ string) {
				if tc.setup != nil {
					tc.setup()
				}
				cmd, ok := commands.Find(tc.cmd)
				if !ok {
					t.Fatalf("could not find command %s", tc.cmd)
				}

				out, err := cmd.Execute(context.Background(), tc.job)
				if err != nil && tc.expectedError != nil {
					stubs.AssertEquals(t, err.Error(), tc.expectedError.Error())
					return
				}

				stubs.Must(t, "cmd erred out", err)
				if tc.expected != "" {
					stubs.AssertEquals(t, tc.expected, out)
				}
				if tc.expectedMatch != "" {
					stubs.AssertMatches(t, tc.expectedMatch, out)
				}
			}))
		})
	}
}

func Test_FilterJobsAudit(t *testing.T) {
	stubs.Must(t, "failed to audit the correct jobs", stubs.WithTmpDB(func(_ string) {
		r1 := request.Request{
			Command:     "command",
			Channel:     "general",
			ChannelID:   "123",
			ChannelLink: "<#123>",
			Username:    "someone",
			Args:        []string{"some", "thing"},
		}
		r2 := request.Request{
			Command:     "command",
			Channel:     "general",
			ChannelID:   "123",
			ChannelLink: "<#123>",
			Username:    "someoneelse",
			Args:        []string{"something", "else"},
		}

		auth.Configure(basicGroups)
		jobs.Create(r1)
		jobs.Create(r2)
		jobs.Create(r1)
		jobs.Create(r1)
		jobs.Create(r2)

		cmd, ok := commands.Find("audit")
		if !ok {
			t.Fatalf("could not find command %s", "audit")
		}

		audit, err := cmd.Execute(context.Background(), jobs.Job{
			Request: request.Request{Args: []string{"-user", "someone"}},
		})
		if err != nil {
			t.Fatalf("Failed to execute audit: %s", err)
		}
		stubs.AssertEquals(t, "*4* - now - *command* by *someone* in *<#123>* - *Running*\n*3* - now - *command* by *someone* in *<#123>* - *Running*\n*1* - now - *command* by *someone* in *<#123>* - *Running*\n", audit)

		limit, err := cmd.Execute(context.Background(), jobs.Job{
			Request: request.Request{Args: []string{"-user", "someone", "-limit", "2"}},
		})
		if err != nil {
			t.Fatalf("Failed to execute audit: %s", err)
		}
		stubs.AssertEquals(t, "*4* - now - *command* by *someone* in *<#123>* - *Running*\n*3* - now - *command* by *someone* in *<#123>* - *Running*\n", limit)
	}))
}
