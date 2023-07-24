#!/usr/bin/env bash

# SPDX-FileCopyrightText: 2023 Christoph Mewes
# SPDX-License-Identifier: MIT

set -euo pipefail

cd $(dirname $0)/../

for filename in contrib/grafana/*.json; do
  tmpfile="$filename.tmp"

  cat "$filename" | \
    jq '(.templating.list[] | select(.type=="query") | .options) = []' | \
    jq '(.templating.list[] | select(.type=="query") | .refresh) = 2' | \
    jq '(.templating.list[] | select(.type=="query") | .current) = {}' | \
    jq '(.templating.list[] | select(.type=="datasource") | .current) = {}' | \
    jq '(.templating.list[] | select(.type=="interval") | .current) = {}' | \
    jq '(.panels[] | select(.scopedVars!=null) | .scopedVars) = {}' | \
    jq '(.panels[].panels?[]? | select(.scopedVars!=null) | .scopedVars) = {}' | \
    jq '(.templating.list[] | select(.type=="datasource") | .hide) = 2' | \
    jq '(.annotations.list) = []' | \
    jq '(.links) = []' | \
    jq '(.refresh) = "1m"' | \
    jq '(.time.from) = "now-1d"' | \
    jq '(.editable) = true' | \
    jq '(.panels[] | select(.type!="row") | .editable) = true' | \
    jq '(.panels[] | select(.type!="row") | .transparent) = true' | \
    jq '(.panels[] | select(.type!="row") | .timeRegions) = []' | \
    jq '(.hideControls) = false' | \
    jq '(.time.to) = "now"' | \
    jq '(.timezone) = ""' | \
    jq '(.graphTooltip) = 1' | \
    jq '(.version) = 1' | \
    jq 'del(.panels[] | select(.repeatPanelId!=null))' | \
    jq 'del(.panels[].panels?[]? | select(.repeatPanelId!=null))' | \
    jq 'del(.id)' | \
    jq 'del(.iteration)' | \
    jq --sort-keys '.' > "$tmpfile"

  mv "$tmpfile" "$filename"
done
