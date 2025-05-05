-- Function to notify schema reload
CREATE OR REPLACE FUNCTION notify_schema_reload(database_name text DEFAULT NULL)
RETURNS void AS $$
BEGIN
  IF database_name IS NULL THEN
    -- Reload all databases
    PERFORM pg_notify('smoothdb', 'reload schema');
  ELSE
    -- Reload specific database
    PERFORM pg_notify('smoothdb', 'reload schema ' || database_name);
  END IF;
END;
$$ LANGUAGE plpgsql;

-- Example usage:
-- SELECT notify_schema_reload(); -- Reload all databases
-- SELECT notify_schema_reload('mydb'); -- Reload specific database

-- You can also use the NOTIFY command directly:
-- NOTIFY smoothdb, 'reload schema'; -- Reload all databases
-- NOTIFY smoothdb, 'reload schema mydb'; -- Reload specific database

-- Optional: Event trigger for automatic schema reload on DDL changes
-- This will automatically notify schema reload when DDL changes occur
CREATE OR REPLACE FUNCTION auto_notify_schema_reload()
RETURNS event_trigger AS $$
BEGIN
  -- Get the current database name
  PERFORM pg_notify('smoothdb', 'reload schema ' || current_database());
END;
$$ LANGUAGE plpgsql;

-- Create the event trigger (commented out by default)
-- Uncomment to enable automatic schema reload on DDL changes
-- CREATE EVENT TRIGGER schema_change_trigger
-- ON ddl_command_end
-- EXECUTE FUNCTION auto_notify_schema_reload();
