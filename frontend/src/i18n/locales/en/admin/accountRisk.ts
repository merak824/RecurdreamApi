export default {
  accountRisk: {
    title: 'Account Risk',
    description: 'Server-side risk control for signup bots, credential stuffing and stolen tokens. Hits are recorded in the audit log (security.risk.*); admins and allowlisted users are only logged, never acted on.',
    reload: 'Reload',
    save: 'Save configuration',
    saving: 'Saving…',
    saved: 'Risk configuration saved; takes effect within ~30s',
    saveFailed: 'Failed to save risk configuration',
    loadFailed: 'Failed to load risk configuration',
    master: {
      title: 'Master switch',
      hint: 'When disabled, all rules stop evaluating and the login guard is also disabled.',
      on: 'Enabled',
      off: 'Disabled',
      rules: 'Registered rules: {rules}'
    },
    fields: {
      action: 'Action',
      windowHours: 'Window (hours)'
    },
    actions: {
      flag: 'Log only',
      ban: 'Suspend account',
      block: 'Reject signup'
    },
    probe: {
      title: 'Signup probing',
      hint: 'If a brand-new account calls the API with a script client (python / curl / empty UA) shortly after signup, treat it as a signup bot.',
      window: 'Observation window (minutes)'
    },
    multi: {
      title: 'Bulk signup per IP',
      hint: 'Reject signups once one IP exceeds the limit inside the window; the "reject" action blocks before the account is created.',
      max: 'Limit per IP'
    },
    session: {
      title: 'Session fingerprint abuse',
      hint: 'Repeated IP+UA fingerprint drift for one account usually means the credential was copied elsewhere.',
      max: 'Max mismatches'
    },
    login: {
      title: 'Login brute-force guard',
      hint: 'Counts failures per IP and per account; exceeds the limit triggers a temporary lock without suspending the account.',
      maxIp: 'Failures per IP',
      ipWindow: 'IP window (minutes)',
      maxAccount: 'Failures per account',
      accountWindow: 'Account window (minutes)',
      lock: 'Lock duration (minutes)'
    },
    events: {
      title: 'Hit records',
      hint: 'Sourced from the audit log, filtered by the security.risk. action prefix.',
      empty: 'No hits yet',
      loadFailed: 'Failed to load hit records',
      filters: {
        all: 'All events',
        banned: 'Suspended',
        blocked: 'Signup blocked',
        loginLocked: 'Login locked',
        flagged: 'Logged only'
      },
      actions: {
        banned: 'Suspended',
        blocked: 'Signup blocked',
        loginLocked: 'Login locked',
        flagged: 'Logged only'
      },
      columns: {
        time: 'Time',
        user: 'User',
        ip: 'Client IP',
        action: 'Action',
        rule: 'Rule',
        reason: 'Reason'
      }
    }
  }
}
