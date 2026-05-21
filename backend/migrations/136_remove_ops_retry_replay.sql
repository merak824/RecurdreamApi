-- Upstream v0.1.129 removes unused Ops retry/replay storage here.
--
-- This fork intentionally keeps the legacy columns/table for this release so a
-- live blue/green deployment can keep the old slot serving while the new slot
-- starts and applies migrations. Dropping these fields before traffic is fully
-- switched can break older code paths that still write/read them.
--
-- After all production instances are safely running a post-v0.1.129 build, the
-- replay storage can be dropped in a separate, low-traffic maintenance migration.

DO $$
BEGIN
  IF to_regclass('public.ops_error_logs') IS NOT NULL THEN
    COMMENT ON TABLE ops_error_logs IS 'Ops error logs (vNext). Legacy retry/replay storage retained for rolling deployment compatibility.';
  END IF;
END $$;
