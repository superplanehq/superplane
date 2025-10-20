--
-- PostgreSQL database dump
--

\restrict qnQvGaqR3gkoOYKHcRDiKR0KTZDqzurLx7D2goWneOaN7pr8Cnlco3ePddc3XRx

-- Dumped from database version 17.5 (Debian 17.5-1.pgdg130+1)
-- Dumped by pg_dump version 17.6 (Debian 17.6-2.pgdg13+1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: uuid-ossp; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;


--
-- Name: EXTENSION "uuid-ossp"; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION "uuid-ossp" IS 'generate universally unique identifiers (UUIDs)';


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: account_providers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.account_providers (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    account_id uuid NOT NULL,
    provider character varying(50) NOT NULL,
    provider_id character varying(255) NOT NULL,
    username character varying(255),
    email character varying(255),
    name character varying(255),
    avatar_url text,
    access_token text,
    refresh_token text,
    token_expires_at timestamp without time zone,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.accounts (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    email character varying(255) NOT NULL,
    name character varying(255) NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: alerts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.alerts (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    canvas_id uuid NOT NULL,
    source_id uuid NOT NULL,
    source_type character varying(255) NOT NULL,
    message text NOT NULL,
    acknowledged boolean DEFAULT false NOT NULL,
    acknowledged_at timestamp with time zone,
    type character varying(50) NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    origin_type character varying(255)
);


--
-- Name: blueprints; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.blueprints (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    organization_id uuid NOT NULL,
    name character varying(128) NOT NULL,
    description text,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    nodes jsonb DEFAULT '[]'::jsonb NOT NULL,
    edges jsonb DEFAULT '[]'::jsonb NOT NULL,
    configuration jsonb DEFAULT '[]'::jsonb NOT NULL,
    output_channels jsonb DEFAULT '[]'::jsonb NOT NULL
);


--
-- Name: canvases; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.canvases (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(128) NOT NULL,
    created_at timestamp without time zone NOT NULL,
    created_by uuid NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    organization_id uuid NOT NULL,
    deleted_at timestamp with time zone,
    description text
);


--
-- Name: casbin_rule; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.casbin_rule (
    id integer NOT NULL,
    ptype character varying(100) NOT NULL,
    v0 character varying(100),
    v1 character varying(100),
    v2 character varying(100),
    v3 character varying(100),
    v4 character varying(100),
    v5 character varying(100)
);


--
-- Name: casbin_rule_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.casbin_rule_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: casbin_rule_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.casbin_rule_id_seq OWNED BY public.casbin_rule.id;


--
-- Name: connection_group_field_set_events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.connection_group_field_set_events (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    connection_group_set_id uuid NOT NULL,
    event_id uuid NOT NULL,
    source_id uuid NOT NULL,
    source_name character varying(128) NOT NULL,
    source_type character varying(64) NOT NULL,
    received_at timestamp without time zone NOT NULL
);


--
-- Name: connection_group_field_sets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.connection_group_field_sets (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    connection_group_id uuid NOT NULL,
    field_set jsonb NOT NULL,
    field_set_hash character(64) NOT NULL,
    state character varying(64) NOT NULL,
    created_at timestamp without time zone NOT NULL,
    timeout integer,
    timeout_behavior character varying(64),
    state_reason character varying(64)
);


--
-- Name: connection_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.connection_groups (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(128) NOT NULL,
    canvas_id uuid NOT NULL,
    created_at timestamp without time zone NOT NULL,
    created_by uuid NOT NULL,
    updated_at timestamp without time zone,
    updated_by uuid,
    spec jsonb DEFAULT '{}'::jsonb NOT NULL,
    description text,
    deleted_at timestamp with time zone
);


--
-- Name: connections; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.connections (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    canvas_id uuid NOT NULL,
    target_id uuid NOT NULL,
    source_id uuid NOT NULL,
    source_name character varying(128) NOT NULL,
    source_type character varying(64) NOT NULL,
    filter_operator character varying(16) NOT NULL,
    filters jsonb NOT NULL,
    target_type character varying(64) DEFAULT 'stage'::character varying NOT NULL
);


--
-- Name: event_rejections; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.event_rejections (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    event_id uuid NOT NULL,
    target_type character varying(64) NOT NULL,
    target_id uuid NOT NULL,
    reason character varying(64) NOT NULL,
    message text,
    rejected_at timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: event_sources; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.event_sources (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    canvas_id uuid NOT NULL,
    name character varying(128) NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    key bytea NOT NULL,
    resource_id uuid,
    state character varying(64) NOT NULL,
    scope character varying(64) NOT NULL,
    description text,
    event_types jsonb DEFAULT '[]'::jsonb NOT NULL,
    deleted_at timestamp with time zone,
    schedule jsonb,
    last_triggered_at timestamp without time zone,
    next_trigger_at timestamp without time zone,
    type character varying(64) DEFAULT 'webhook'::character varying NOT NULL
);


--
-- Name: events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.events (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    source_id uuid NOT NULL,
    canvas_id uuid NOT NULL,
    source_name character varying(128) NOT NULL,
    source_type character varying(64) NOT NULL,
    received_at timestamp without time zone NOT NULL,
    raw jsonb NOT NULL,
    state character varying(64) NOT NULL,
    headers jsonb DEFAULT '{}'::jsonb NOT NULL,
    type character varying(128) NOT NULL,
    state_reason character varying(64),
    state_message text,
    created_by uuid
);


--
-- Name: execution_resources; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.execution_resources (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    external_id character varying(128) NOT NULL,
    type character varying(64) NOT NULL,
    stage_id uuid NOT NULL,
    execution_id uuid NOT NULL,
    parent_resource_id uuid NOT NULL,
    state character varying(64) NOT NULL,
    result character varying(64) NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    last_polled_at timestamp without time zone
);


--
-- Name: group_metadata; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.group_metadata (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    group_name character varying(255) NOT NULL,
    domain_type character varying(50) NOT NULL,
    domain_id character varying(255) NOT NULL,
    display_name character varying(255) NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: integrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.integrations (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(128) NOT NULL,
    domain_type character varying(64) NOT NULL,
    domain_id uuid NOT NULL,
    created_at timestamp without time zone NOT NULL,
    created_by uuid NOT NULL,
    updated_at timestamp without time zone,
    type character varying(64) NOT NULL,
    url character varying(256) NOT NULL,
    auth_type character varying(64) NOT NULL,
    auth jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: organization_invitations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.organization_invitations (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    organization_id uuid NOT NULL,
    email character varying(255) NOT NULL,
    invited_by uuid NOT NULL,
    state character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    canvas_ids jsonb DEFAULT '[]'::jsonb
);


--
-- Name: organizations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.organizations (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(255) NOT NULL,
    allowed_providers jsonb DEFAULT '[]'::jsonb NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    deleted_at timestamp without time zone,
    description text DEFAULT ''::text
);


--
-- Name: resources; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.resources (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    external_id character varying(128) NOT NULL,
    type character varying(64) NOT NULL,
    name character varying(128) NOT NULL,
    integration_id uuid NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone,
    parent_id uuid
);


--
-- Name: role_metadata; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.role_metadata (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    role_name character varying(255) NOT NULL,
    domain_type character varying(50) NOT NULL,
    domain_id character varying(255) NOT NULL,
    display_name character varying(255) NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


--
-- Name: secrets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.secrets (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(128) NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    created_by uuid NOT NULL,
    provider character varying(64) NOT NULL,
    data bytea NOT NULL,
    domain_type character varying(64) NOT NULL,
    domain_id character varying(64) NOT NULL
);


--
-- Name: stage_event_approvals; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.stage_event_approvals (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    stage_event_id uuid NOT NULL,
    approved_at timestamp without time zone NOT NULL,
    approved_by uuid NOT NULL
);


--
-- Name: stage_events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.stage_events (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    stage_id uuid NOT NULL,
    event_id uuid NOT NULL,
    source_id uuid NOT NULL,
    source_name character varying(128) NOT NULL,
    source_type character varying(64) NOT NULL,
    state character varying(64) NOT NULL,
    state_reason character varying(64),
    created_at timestamp without time zone NOT NULL,
    inputs jsonb DEFAULT '{}'::jsonb NOT NULL,
    name text,
    discarded_by uuid,
    discarded_at timestamp without time zone
);


--
-- Name: stage_executions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.stage_executions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    canvas_id uuid NOT NULL,
    stage_id uuid NOT NULL,
    stage_event_id uuid NOT NULL,
    state character varying(64) NOT NULL,
    result character varying(64) NOT NULL,
    outputs jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    started_at timestamp without time zone,
    finished_at timestamp without time zone,
    cancelled_at timestamp without time zone,
    cancelled_by uuid,
    result_reason character varying(64),
    result_message text
);


--
-- Name: stages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.stages (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(128) NOT NULL,
    canvas_id uuid NOT NULL,
    created_at timestamp without time zone NOT NULL,
    created_by uuid NOT NULL,
    updated_at timestamp without time zone,
    updated_by uuid,
    executor_spec jsonb DEFAULT '{}'::jsonb NOT NULL,
    conditions jsonb,
    inputs jsonb DEFAULT '[]'::jsonb NOT NULL,
    outputs jsonb DEFAULT '[]'::jsonb NOT NULL,
    input_mappings jsonb DEFAULT '[]'::jsonb NOT NULL,
    secrets jsonb DEFAULT '[]'::jsonb NOT NULL,
    executor_type character varying(64) NOT NULL,
    resource_id uuid,
    description text,
    executor_name text,
    deleted_at timestamp with time zone,
    dry_run boolean DEFAULT false
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    account_id uuid NOT NULL,
    name character varying(255),
    email character varying(255),
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    deleted_at timestamp without time zone,
    organization_id uuid NOT NULL,
    token_hash character varying(250)
);


--
-- Name: workflow_events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflow_events (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    workflow_id uuid NOT NULL,
    node_id character varying(128),
    channel character varying(64),
    data jsonb NOT NULL,
    state character varying(32) NOT NULL,
    execution_id uuid,
    created_at timestamp without time zone NOT NULL
);


--
-- Name: workflow_node_execution_requests; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflow_node_execution_requests (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    workflow_id uuid NOT NULL,
    execution_id uuid NOT NULL,
    state character varying(32) NOT NULL,
    type character varying(32) NOT NULL,
    spec jsonb NOT NULL,
    run_at timestamp without time zone NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);


--
-- Name: workflow_node_executions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflow_node_executions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    workflow_id uuid NOT NULL,
    node_id character varying(128) NOT NULL,
    root_event_id uuid NOT NULL,
    event_id uuid NOT NULL,
    previous_execution_id uuid,
    parent_execution_id uuid,
    state character varying(32) NOT NULL,
    result character varying(32),
    result_reason character varying(128),
    result_message text,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    configuration jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);


--
-- Name: workflow_node_queue_items; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflow_node_queue_items (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    workflow_id uuid NOT NULL,
    node_id character varying(128) NOT NULL,
    root_event_id uuid NOT NULL,
    event_id uuid NOT NULL,
    created_at timestamp without time zone NOT NULL
);


--
-- Name: workflow_nodes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflow_nodes (
    workflow_id uuid NOT NULL,
    node_id character varying(128) NOT NULL,
    name character varying(128) NOT NULL,
    state character varying(32) NOT NULL,
    type character varying(32) NOT NULL,
    ref jsonb NOT NULL,
    configuration jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);


--
-- Name: workflows; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflows (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    organization_id uuid NOT NULL,
    name character varying(128) NOT NULL,
    description text,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    edges jsonb DEFAULT '[]'::jsonb NOT NULL
);


--
-- Name: casbin_rule id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.casbin_rule ALTER COLUMN id SET DEFAULT nextval('public.casbin_rule_id_seq'::regclass);


--
-- Name: account_providers account_providers_account_id_provider_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_providers
    ADD CONSTRAINT account_providers_account_id_provider_key UNIQUE (account_id, provider);


--
-- Name: account_providers account_providers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_providers
    ADD CONSTRAINT account_providers_pkey PRIMARY KEY (id);


--
-- Name: account_providers account_providers_provider_provider_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_providers
    ADD CONSTRAINT account_providers_provider_provider_id_key UNIQUE (provider, provider_id);


--
-- Name: accounts accounts_email_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_email_key UNIQUE (email);


--
-- Name: accounts accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_pkey PRIMARY KEY (id);


--
-- Name: alerts alerts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.alerts
    ADD CONSTRAINT alerts_pkey PRIMARY KEY (id);


--
-- Name: blueprints blueprints_organization_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.blueprints
    ADD CONSTRAINT blueprints_organization_id_name_key UNIQUE (organization_id, name);


--
-- Name: blueprints blueprints_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.blueprints
    ADD CONSTRAINT blueprints_pkey PRIMARY KEY (id);


--
-- Name: canvases canvases_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.canvases
    ADD CONSTRAINT canvases_pkey PRIMARY KEY (id);


--
-- Name: casbin_rule casbin_rule_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.casbin_rule
    ADD CONSTRAINT casbin_rule_pkey PRIMARY KEY (id);


--
-- Name: connection_group_field_set_events connection_group_field_set_events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.connection_group_field_set_events
    ADD CONSTRAINT connection_group_field_set_events_pkey PRIMARY KEY (id);


--
-- Name: connection_group_field_sets connection_group_field_sets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.connection_group_field_sets
    ADD CONSTRAINT connection_group_field_sets_pkey PRIMARY KEY (id);


--
-- Name: connection_groups connection_groups_canvas_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.connection_groups
    ADD CONSTRAINT connection_groups_canvas_id_name_key UNIQUE (canvas_id, name);


--
-- Name: connection_groups connection_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.connection_groups
    ADD CONSTRAINT connection_groups_pkey PRIMARY KEY (id);


--
-- Name: connections connections_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.connections
    ADD CONSTRAINT connections_pkey PRIMARY KEY (id);


--
-- Name: connections connections_target_id_source_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.connections
    ADD CONSTRAINT connections_target_id_source_id_key UNIQUE (target_id, source_id);


--
-- Name: event_rejections event_rejections_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_rejections
    ADD CONSTRAINT event_rejections_pkey PRIMARY KEY (id);


--
-- Name: event_sources event_sources_canvas_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_sources
    ADD CONSTRAINT event_sources_canvas_id_name_key UNIQUE (canvas_id, name);


--
-- Name: event_sources event_sources_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_sources
    ADD CONSTRAINT event_sources_pkey PRIMARY KEY (id);


--
-- Name: events events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.events
    ADD CONSTRAINT events_pkey PRIMARY KEY (id);


--
-- Name: execution_resources execution_resources_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.execution_resources
    ADD CONSTRAINT execution_resources_pkey PRIMARY KEY (id);


--
-- Name: group_metadata group_metadata_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_metadata
    ADD CONSTRAINT group_metadata_pkey PRIMARY KEY (id);


--
-- Name: integrations integrations_domain_type_domain_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.integrations
    ADD CONSTRAINT integrations_domain_type_domain_id_name_key UNIQUE (domain_type, domain_id, name);


--
-- Name: integrations integrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.integrations
    ADD CONSTRAINT integrations_pkey PRIMARY KEY (id);


--
-- Name: organization_invitations organization_invitations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_invitations
    ADD CONSTRAINT organization_invitations_pkey PRIMARY KEY (id);


--
-- Name: organizations organizations_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organizations
    ADD CONSTRAINT organizations_name_key UNIQUE (name);


--
-- Name: organizations organizations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organizations
    ADD CONSTRAINT organizations_pkey PRIMARY KEY (id);


--
-- Name: resources resources_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.resources
    ADD CONSTRAINT resources_pkey PRIMARY KEY (id);


--
-- Name: role_metadata role_metadata_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_metadata
    ADD CONSTRAINT role_metadata_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: secrets secrets_domain_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.secrets
    ADD CONSTRAINT secrets_domain_id_name_key UNIQUE (domain_type, domain_id, name);


--
-- Name: secrets secrets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.secrets
    ADD CONSTRAINT secrets_pkey PRIMARY KEY (id);


--
-- Name: stage_event_approvals stage_event_approvals_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage_event_approvals
    ADD CONSTRAINT stage_event_approvals_pkey PRIMARY KEY (id);


--
-- Name: stage_event_approvals stage_event_approvals_stage_event_id_approved_by_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage_event_approvals
    ADD CONSTRAINT stage_event_approvals_stage_event_id_approved_by_key UNIQUE (stage_event_id, approved_by);


--
-- Name: stage_events stage_events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage_events
    ADD CONSTRAINT stage_events_pkey PRIMARY KEY (id);


--
-- Name: stage_executions stage_executions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage_executions
    ADD CONSTRAINT stage_executions_pkey PRIMARY KEY (id);


--
-- Name: stages stages_canvas_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stages
    ADD CONSTRAINT stages_canvas_id_name_key UNIQUE (canvas_id, name);


--
-- Name: stages stages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stages
    ADD CONSTRAINT stages_pkey PRIMARY KEY (id);


--
-- Name: canvases unique_canvas_in_organization; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.canvases
    ADD CONSTRAINT unique_canvas_in_organization UNIQUE (organization_id, name);


--
-- Name: users unique_user_in_organization; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT unique_user_in_organization UNIQUE (organization_id, account_id, email);


--
-- Name: group_metadata uq_group_metadata_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_metadata
    ADD CONSTRAINT uq_group_metadata_key UNIQUE (group_name, domain_type, domain_id);


--
-- Name: role_metadata uq_role_metadata_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_metadata
    ADD CONSTRAINT uq_role_metadata_key UNIQUE (role_name, domain_type, domain_id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: workflow_events workflow_events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_events
    ADD CONSTRAINT workflow_events_pkey PRIMARY KEY (id);


--
-- Name: workflow_node_execution_requests workflow_node_execution_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_execution_requests
    ADD CONSTRAINT workflow_node_execution_requests_pkey PRIMARY KEY (id);


--
-- Name: workflow_node_executions workflow_node_executions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_executions
    ADD CONSTRAINT workflow_node_executions_pkey PRIMARY KEY (id);


--
-- Name: workflow_node_queue_items workflow_node_queue_items_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_queue_items
    ADD CONSTRAINT workflow_node_queue_items_pkey PRIMARY KEY (id);


--
-- Name: workflow_nodes workflow_nodes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_nodes
    ADD CONSTRAINT workflow_nodes_pkey PRIMARY KEY (workflow_id, node_id);


--
-- Name: workflows workflows_organization_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflows
    ADD CONSTRAINT workflows_organization_id_name_key UNIQUE (organization_id, name);


--
-- Name: workflows workflows_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflows
    ADD CONSTRAINT workflows_pkey PRIMARY KEY (id);


--
-- Name: idx_account_providers_account_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_account_providers_account_id ON public.account_providers USING btree (account_id);


--
-- Name: idx_account_providers_provider; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_account_providers_provider ON public.account_providers USING btree (provider);


--
-- Name: idx_alerts_canvas_acknowledged; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alerts_canvas_acknowledged ON public.alerts USING btree (canvas_id, acknowledged);


--
-- Name: idx_alerts_canvas_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alerts_canvas_id ON public.alerts USING btree (canvas_id);


--
-- Name: idx_alerts_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_alerts_created_at ON public.alerts USING btree (created_at DESC);


--
-- Name: idx_blueprints_organization_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_blueprints_organization_id ON public.blueprints USING btree (organization_id);


--
-- Name: idx_canvases_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_canvases_deleted_at ON public.canvases USING btree (deleted_at);


--
-- Name: idx_casbin_rule_ptype; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_casbin_rule_ptype ON public.casbin_rule USING btree (ptype);


--
-- Name: idx_casbin_rule_v0; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_casbin_rule_v0 ON public.casbin_rule USING btree (v0);


--
-- Name: idx_casbin_rule_v1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_casbin_rule_v1 ON public.casbin_rule USING btree (v1);


--
-- Name: idx_casbin_rule_v2; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_casbin_rule_v2 ON public.casbin_rule USING btree (v2);


--
-- Name: idx_connection_groups_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_connection_groups_deleted_at ON public.connection_groups USING btree (deleted_at);


--
-- Name: idx_event_rejections_event_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_event_rejections_event_id ON public.event_rejections USING btree (event_id);


--
-- Name: idx_event_rejections_target; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_event_rejections_target ON public.event_rejections USING btree (target_type, target_id);


--
-- Name: idx_event_sources_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_event_sources_deleted_at ON public.event_sources USING btree (deleted_at);


--
-- Name: idx_event_sources_next_trigger_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_event_sources_next_trigger_at ON public.event_sources USING btree (next_trigger_at);


--
-- Name: idx_group_metadata_lookup; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_group_metadata_lookup ON public.group_metadata USING btree (group_name, domain_type, domain_id);


--
-- Name: idx_node_execution_requests_state_run_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_node_execution_requests_state_run_at ON public.workflow_node_execution_requests USING btree (state, run_at) WHERE ((state)::text = 'pending'::text);


--
-- Name: idx_organizations_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_organizations_deleted_at ON public.organizations USING btree (deleted_at);


--
-- Name: idx_role_metadata_lookup; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_role_metadata_lookup ON public.role_metadata USING btree (role_name, domain_type, domain_id);


--
-- Name: idx_stage_events_stage_id_state_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_stage_events_stage_id_state_created_at ON public.stage_events USING btree (stage_id, state, created_at);


--
-- Name: idx_stage_executions_id_stage_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_stage_executions_id_stage_id ON public.stage_executions USING btree (id, stage_id);


--
-- Name: idx_stage_executions_stage_id_state_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_stage_executions_stage_id_state_created_at ON public.stage_executions USING btree (stage_id, state, created_at DESC);


--
-- Name: idx_stages_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_stages_deleted_at ON public.stages USING btree (deleted_at);


--
-- Name: idx_workflow_events_workflow_node_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_events_workflow_node_id ON public.workflow_events USING btree (workflow_id, node_id);


--
-- Name: idx_workflow_node_executions_workflow_node_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_executions_workflow_node_id ON public.workflow_node_executions USING btree (workflow_id, node_id);


--
-- Name: idx_workflow_nodes_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_nodes_state ON public.workflow_nodes USING btree (state);


--
-- Name: idx_workflows_organization_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflows_organization_id ON public.workflows USING btree (organization_id);


--
-- Name: uix_event_sources_canvas; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX uix_event_sources_canvas ON public.event_sources USING btree (canvas_id);


--
-- Name: uix_events_canvas; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX uix_events_canvas ON public.events USING btree (canvas_id);


--
-- Name: uix_events_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX uix_events_source ON public.events USING btree (source_id);


--
-- Name: uix_stage_event_approvals_events; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX uix_stage_event_approvals_events ON public.stage_event_approvals USING btree (stage_event_id);


--
-- Name: uix_stage_events_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX uix_stage_events_source ON public.stage_events USING btree (source_id);


--
-- Name: uix_stage_events_stage; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX uix_stage_events_stage ON public.stage_events USING btree (stage_id);


--
-- Name: uix_stage_executions_events; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX uix_stage_executions_events ON public.stage_executions USING btree (stage_event_id);


--
-- Name: uix_stage_executions_stage; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX uix_stage_executions_stage ON public.stage_executions USING btree (stage_id);


--
-- Name: uix_stages_canvas; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX uix_stages_canvas ON public.stages USING btree (canvas_id);


--
-- Name: account_providers account_providers_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_providers
    ADD CONSTRAINT account_providers_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id);


--
-- Name: canvases canvases_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.canvases
    ADD CONSTRAINT canvases_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id);


--
-- Name: connection_group_field_set_events connection_group_field_set_events_connection_group_set_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.connection_group_field_set_events
    ADD CONSTRAINT connection_group_field_set_events_connection_group_set_id_fkey FOREIGN KEY (connection_group_set_id) REFERENCES public.connection_group_field_sets(id);


--
-- Name: connection_group_field_sets connection_group_field_sets_connection_group_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.connection_group_field_sets
    ADD CONSTRAINT connection_group_field_sets_connection_group_id_fkey FOREIGN KEY (connection_group_id) REFERENCES public.connection_groups(id);


--
-- Name: connection_groups connection_groups_canvas_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.connection_groups
    ADD CONSTRAINT connection_groups_canvas_id_fkey FOREIGN KEY (canvas_id) REFERENCES public.canvases(id);


--
-- Name: event_rejections event_rejections_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_rejections
    ADD CONSTRAINT event_rejections_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.events(id);


--
-- Name: event_sources event_sources_canvas_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_sources
    ADD CONSTRAINT event_sources_canvas_id_fkey FOREIGN KEY (canvas_id) REFERENCES public.canvases(id);


--
-- Name: event_sources event_sources_resource_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_sources
    ADD CONSTRAINT event_sources_resource_id_fkey FOREIGN KEY (resource_id) REFERENCES public.resources(id);


--
-- Name: execution_resources execution_resources_execution_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.execution_resources
    ADD CONSTRAINT execution_resources_execution_id_fkey FOREIGN KEY (execution_id) REFERENCES public.stage_executions(id);


--
-- Name: execution_resources execution_resources_parent_resource_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.execution_resources
    ADD CONSTRAINT execution_resources_parent_resource_id_fkey FOREIGN KEY (parent_resource_id) REFERENCES public.resources(id);


--
-- Name: organization_invitations organization_invitations_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_invitations
    ADD CONSTRAINT organization_invitations_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: resources resources_integration_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.resources
    ADD CONSTRAINT resources_integration_id_fkey FOREIGN KEY (integration_id) REFERENCES public.integrations(id);


--
-- Name: connections stage_connections_canvas_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.connections
    ADD CONSTRAINT stage_connections_canvas_id_fkey FOREIGN KEY (canvas_id) REFERENCES public.canvases(id);


--
-- Name: stage_event_approvals stage_event_approvals_stage_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage_event_approvals
    ADD CONSTRAINT stage_event_approvals_stage_event_id_fkey FOREIGN KEY (stage_event_id) REFERENCES public.stage_events(id) ON DELETE CASCADE;


--
-- Name: stage_events stage_events_stage_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage_events
    ADD CONSTRAINT stage_events_stage_id_fkey FOREIGN KEY (stage_id) REFERENCES public.stages(id);


--
-- Name: stage_executions stage_executions_canvas_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage_executions
    ADD CONSTRAINT stage_executions_canvas_id_fkey FOREIGN KEY (canvas_id) REFERENCES public.canvases(id);


--
-- Name: stage_executions stage_executions_stage_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage_executions
    ADD CONSTRAINT stage_executions_stage_event_id_fkey FOREIGN KEY (stage_event_id) REFERENCES public.stage_events(id);


--
-- Name: stage_executions stage_executions_stage_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage_executions
    ADD CONSTRAINT stage_executions_stage_id_fkey FOREIGN KEY (stage_id) REFERENCES public.stages(id);


--
-- Name: stages stages_canvas_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stages
    ADD CONSTRAINT stages_canvas_id_fkey FOREIGN KEY (canvas_id) REFERENCES public.canvases(id);


--
-- Name: users users_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id);


--
-- Name: users users_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id);


--
-- Name: workflow_events workflow_events_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_events
    ADD CONSTRAINT workflow_events_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id) ON DELETE CASCADE;


--
-- Name: workflow_node_execution_requests workflow_node_execution_requests_execution_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_execution_requests
    ADD CONSTRAINT workflow_node_execution_requests_execution_id_fkey FOREIGN KEY (execution_id) REFERENCES public.workflow_node_executions(id) ON DELETE CASCADE;


--
-- Name: workflow_node_execution_requests workflow_node_execution_requests_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_execution_requests
    ADD CONSTRAINT workflow_node_execution_requests_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id) ON DELETE CASCADE;


--
-- Name: workflow_node_executions workflow_node_executions_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_executions
    ADD CONSTRAINT workflow_node_executions_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.workflow_events(id) ON DELETE CASCADE;


--
-- Name: workflow_node_executions workflow_node_executions_parent_execution_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_executions
    ADD CONSTRAINT workflow_node_executions_parent_execution_id_fkey FOREIGN KEY (parent_execution_id) REFERENCES public.workflow_node_executions(id) ON DELETE CASCADE;


--
-- Name: workflow_node_executions workflow_node_executions_previous_execution_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_executions
    ADD CONSTRAINT workflow_node_executions_previous_execution_id_fkey FOREIGN KEY (previous_execution_id) REFERENCES public.workflow_node_executions(id) ON DELETE CASCADE;


--
-- Name: workflow_node_executions workflow_node_executions_root_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_executions
    ADD CONSTRAINT workflow_node_executions_root_event_id_fkey FOREIGN KEY (root_event_id) REFERENCES public.workflow_events(id) ON DELETE CASCADE;


--
-- Name: workflow_node_executions workflow_node_executions_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_executions
    ADD CONSTRAINT workflow_node_executions_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id) ON DELETE CASCADE;


--
-- Name: workflow_node_queue_items workflow_node_queue_items_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_queue_items
    ADD CONSTRAINT workflow_node_queue_items_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.workflow_events(id) ON DELETE CASCADE;


--
-- Name: workflow_node_queue_items workflow_node_queue_items_root_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_queue_items
    ADD CONSTRAINT workflow_node_queue_items_root_event_id_fkey FOREIGN KEY (root_event_id) REFERENCES public.workflow_events(id) ON DELETE CASCADE;


--
-- Name: workflow_node_queue_items workflow_node_queue_items_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_queue_items
    ADD CONSTRAINT workflow_node_queue_items_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id) ON DELETE CASCADE;


--
-- Name: workflow_nodes workflow_nodes_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_nodes
    ADD CONSTRAINT workflow_nodes_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

\unrestrict qnQvGaqR3gkoOYKHcRDiKR0KTZDqzurLx7D2goWneOaN7pr8Cnlco3ePddc3XRx

--
-- PostgreSQL database dump
--

\restrict ahkKfch15PvapnALWQNoJcAVLIn5a5Ry6se3spA9XgnwXZmEcnhLrweS7pJTXUd

-- Dumped from database version 17.5 (Debian 17.5-1.pgdg130+1)
-- Dumped by pg_dump version 17.6 (Debian 17.6-2.pgdg13+1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Data for Name: schema_migrations; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.schema_migrations (version, dirty) FROM stdin;
20251020120155	f
\.


--
-- PostgreSQL database dump complete
--

\unrestrict ahkKfch15PvapnALWQNoJcAVLIn5a5Ry6se3spA9XgnwXZmEcnhLrweS7pJTXUd

