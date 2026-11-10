import overview from './overview'
import channels from './channels'
import accounts from './accounts'
import resources from './resources'
import ops from './ops'
import settings from './settings'
import audit from './audit'
import accountRisk from './accountRisk'
import promptAudit from './promptAudit'
import plugins from './plugins'
import redPacket from './redPacket'

export default {
  ...overview,
  ...channels,
  ...accounts,
  ...resources,
  ...ops,
  ...settings,
  ...audit,
  ...accountRisk,
  ...promptAudit,
  ...plugins,
  ...redPacket,
}
