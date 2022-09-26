/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"
)

type PRDetails struct {
	Number         int
	MergeCommitSHA string
	HeadSHA        string
	State          string
	ClosedAt       time.Time
}

func GetGHClient(ghToken string) *github.Client {

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc)

}

func (r *PREphemeralEnvControllerReconciler) GetActivePullRequests() ([]PRDetails, error) {

	ghClient := GetGHClient(r.GHPATToken)
	if ghClient == nil {
		return nil, fmt.Errorf("failed to get github client")
	}

	var activePullRequests []PRDetails

	opts := &github.PullRequestListOptions{

		State: "open",
	}

	pullRequests, _, err := ghClient.PullRequests.List(context.Background(), r.GHPRRepo.User, r.GHPRRepo.Repo, opts)

	if err != nil {
		return nil, err
	}

	for _, pullRequest := range pullRequests {
		if !pullRequest.GetMerged() {
			prD := PRDetails{
				Number:         pullRequest.GetNumber(),
				MergeCommitSHA: pullRequest.GetMergeCommitSHA(),
				HeadSHA:        pullRequest.GetHead().GetSHA(),
				State:          pullRequest.GetState(),
				ClosedAt:       pullRequest.GetClosedAt(),
			}
			activePullRequests = append(activePullRequests, prD)
		}
	}

	return activePullRequests, nil
}

func (r *PREphemeralEnvControllerReconciler) UpdatePRStatus(context context.Context, prNumber int, prSHA string, status string, description string) error {

	ghClient := GetGHClient(r.GHPATToken)
	if ghClient == nil {
		return fmt.Errorf("failed to get github client")
	}

	repoStatus := &github.RepoStatus{
		State:       &status,
		Description: &description,
	}

	_, _, err := ghClient.Repositories.CreateStatus(context, r.GHPRRepo.User, r.GHPRRepo.Repo, prSHA, repoStatus)

	if err != nil {
		return err
	}

	return nil
}
