-- Advance every serial/identity sequence and every sequence referenced by a
-- column default in the current PostgreSQL database to at least that column's
-- largest stored value. Existing rows and sequences already ahead are unchanged.
DO $$
DECLARE
    sequence_column RECORD;
    max_id BIGINT;
    current_value BIGINT;
    sequence_called BOOLEAN;
    sequence_ref REGCLASS;
BEGIN
    FOR sequence_column IN
        SELECT
            sequence_ns.nspname AS sequence_schema,
            sequence_class.relname AS sequence_name,
            table_ns.nspname AS table_schema,
            table_class.relname AS table_name,
            table_column.attname AS column_name
        FROM pg_catalog.pg_class AS sequence_class
        JOIN pg_catalog.pg_namespace AS sequence_ns
            ON sequence_ns.oid = sequence_class.relnamespace
        JOIN pg_catalog.pg_depend AS ownership
            ON ownership.objid = sequence_class.oid
           AND ownership.classid = 'pg_catalog.pg_class'::REGCLASS
           AND ownership.refclassid = 'pg_catalog.pg_class'::REGCLASS
           AND ownership.deptype IN ('a', 'i')
        JOIN pg_catalog.pg_class AS table_class
            ON table_class.oid = ownership.refobjid
           AND table_class.relkind IN ('r', 'p')
        JOIN pg_catalog.pg_namespace AS table_ns
            ON table_ns.oid = table_class.relnamespace
        JOIN pg_catalog.pg_attribute AS table_column
            ON table_column.attrelid = table_class.oid
           AND table_column.attnum = ownership.refobjsubid
           AND NOT table_column.attisdropped
        WHERE sequence_class.relkind = 'S'
          AND sequence_ns.nspname <> 'information_schema'
          AND sequence_ns.nspname !~ '^pg_'

        UNION

        SELECT
            sequence_ns.nspname AS sequence_schema,
            sequence_class.relname AS sequence_name,
            table_ns.nspname AS table_schema,
            table_class.relname AS table_name,
            table_column.attname AS column_name
        FROM pg_catalog.pg_class AS sequence_class
        JOIN pg_catalog.pg_namespace AS sequence_ns
            ON sequence_ns.oid = sequence_class.relnamespace
        JOIN pg_catalog.pg_depend AS default_reference
            ON default_reference.objid = sequence_class.oid
           AND default_reference.classid = 'pg_catalog.pg_class'::REGCLASS
           AND default_reference.refclassid = 'pg_catalog.pg_attrdef'::REGCLASS
           AND default_reference.deptype = 'n'
        JOIN pg_catalog.pg_attrdef AS column_default
            ON column_default.oid = default_reference.refobjid
        JOIN pg_catalog.pg_class AS table_class
            ON table_class.oid = column_default.adrelid
           AND table_class.relkind IN ('r', 'p')
        JOIN pg_catalog.pg_namespace AS table_ns
            ON table_ns.oid = table_class.relnamespace
        JOIN pg_catalog.pg_attribute AS table_column
            ON table_column.attrelid = column_default.adrelid
           AND table_column.attnum = column_default.adnum
           AND NOT table_column.attisdropped
        WHERE sequence_class.relkind = 'S'
          AND sequence_ns.nspname <> 'information_schema'
          AND sequence_ns.nspname !~ '^pg_'
    LOOP
        sequence_ref := FORMAT('%I.%I', sequence_column.sequence_schema, sequence_column.sequence_name)::REGCLASS;

        EXECUTE FORMAT(
            'SELECT MAX(%I)::BIGINT FROM %I.%I',
            sequence_column.column_name,
            sequence_column.table_schema,
            sequence_column.table_name
        ) INTO max_id;

        IF max_id IS NULL THEN
            CONTINUE;
        END IF;

        EXECUTE FORMAT('SELECT last_value, is_called FROM %s', sequence_ref)
            INTO current_value, sequence_called;

        IF NOT sequence_called OR max_id > current_value THEN
            PERFORM pg_catalog.setval(sequence_ref, max_id, TRUE);
        END IF;
    END LOOP;
END $$;
