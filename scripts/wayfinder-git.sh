# shellcheck shell=bash
# Sourced helper: git access to the Wayfinder source repo.
#
# SECURITY: WAYFINDER_READ_TOKEN reaches git ONLY through GIT_CONFIG_* environment
# variables exported inside a subshell around each git call. It is never put on
# argv (visible in `ps`), in the URL, in a file, or in any message.

WF_URL="${WAYFINDER_GIT_URL:-https://github.com/appvia/wayfinder.git}"

wf_redacted_url() {
  printf '%s\n' "$WF_URL" | sed -E 's#://[^/@]*@#://#'
}

wf_git() {
  if [ -n "${WAYFINDER_READ_TOKEN:-}" ]; then
    (
      GIT_CONFIG_VALUE_0="AUTHORIZATION: basic $(printf 'x-access-token:%s' "$WAYFINDER_READ_TOKEN" | base64 | tr -d '\n')"
      export GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=http.https://github.com/.extraheader GIT_CONFIG_VALUE_0
      unset WAYFINDER_READ_TOKEN
      git "$@"
    )
  else
    git "$@"
  fi
}
