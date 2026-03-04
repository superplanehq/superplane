--
-- PostgreSQL database dump
--

\restrict abcdef123

-- Dumped from database version 17.5 (Debian 17.5-1.pgdg130+1)
-- Dumped by pg_dump version 17.8 (Ubuntu 17.8-1.pgdg22.04+1)

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
-- Name: account_password_auth; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.account_password_auth (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    account_id uuid NOT NULL,
    password_hash character varying(255) NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


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
-- Name: app_installation_requests; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.app_installation_requests (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    app_installation_id uuid NOT NULL,
    state character varying(32) NOT NULL,
    type character varying(32) NOT NULL,
    run_at timestamp without time zone NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    spec jsonb
);


--
-- Name: app_installation_secrets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.app_installation_secrets (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    organization_id uuid NOT NULL,
    installation_id uuid NOT NULL,
    name character varying(64) NOT NULL,
    value bytea NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);


--
-- Name: app_installation_subscriptions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.app_installation_subscriptions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    installation_id uuid NOT NULL,
    workflow_id uuid NOT NULL,
    node_id character varying(128) NOT NULL,
    configuration jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);


--
-- Name: app_installations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.app_installations (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    organization_id uuid NOT NULL,
    app_name character varying(255) NOT NULL,
    installation_name character varying(255) NOT NULL,
    state character varying(32) NOT NULL,
    state_description character varying(1024),
    configuration jsonb DEFAULT '{}'::jsonb NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    browser_action jsonb,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    deleted_at timestamp with time zone
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
    output_channels jsonb DEFAULT '[]'::jsonb NOT NULL,
    icon character varying(32),
    color character varying(32),
    created_by uuid
);


--
-- Name: canvas_memories; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.canvas_memories (
    canvas_id uuid NOT NULL,
    namespace text NOT NULL,
    "values" jsonb NOT NULL,
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
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
-- Name: data_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.data_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


--
-- Name: email_settings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.email_settings (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    provider character varying(50) NOT NULL,
    smtp_host character varying(255),
    smtp_port integer,
    smtp_username character varying(255),
    smtp_password bytea,
    smtp_from_name character varying(255),
    smtp_from_email character varying(255),
    smtp_use_tls boolean DEFAULT false NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
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
-- Name: installation_metadata; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.installation_metadata (
    id integer NOT NULL,
    installation_id character varying(64) NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT installation_metadata_singleton CHECK ((id = 1))
);


--
-- Name: organization_agent_settings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.organization_agent_settings (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    organization_id uuid NOT NULL,
    agent_mode_enabled boolean DEFAULT false NOT NULL,
    openai_api_key_ciphertext bytea,
    openai_key_encryption_key_id character varying(255),
    openai_key_last4 character varying(8),
    openai_key_status character varying(32) DEFAULT 'not_configured'::character varying NOT NULL,
    openai_key_validated_at timestamp without time zone,
    openai_key_validation_error text,
    updated_by uuid,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT organization_agent_settings_openai_key_status_check CHECK (((openai_key_status)::text = ANY ((ARRAY['not_configured'::character varying, 'valid'::character varying, 'invalid'::character varying, 'unchecked'::character varying])::text[])))
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
-- Name: organization_invite_links; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.organization_invite_links (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    organization_id uuid NOT NULL,
    token uuid NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
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
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    account_id uuid,
    name character varying(255),
    email character varying(255),
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    deleted_at timestamp without time zone,
    organization_id uuid NOT NULL,
    token_hash character varying(250),
    type character varying(50) DEFAULT 'human'::character varying NOT NULL,
    description text,
    created_by uuid
);


--
-- Name: webhooks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.webhooks (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    state character varying(32) NOT NULL,
    secret bytea NOT NULL,
    configuration jsonb DEFAULT '{}'::jsonb NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    deleted_at timestamp without time zone,
    retry_count integer DEFAULT 0 NOT NULL,
    max_retries integer DEFAULT 3 NOT NULL,
    app_installation_id uuid
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
    created_at timestamp without time zone NOT NULL,
    custom_name text
);


--
-- Name: workflow_node_execution_kvs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflow_node_execution_kvs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    execution_id uuid NOT NULL,
    key text NOT NULL,
    value text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    workflow_id uuid NOT NULL,
    node_id character varying(128) NOT NULL
);


--
-- Name: workflow_node_executions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflow_node_executions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    workflow_id uuid NOT NULL,
    node_id character varying(128) NOT NULL,
    root_event_id uuid,
    event_id uuid,
    previous_execution_id uuid,
    parent_execution_id uuid,
    state character varying(32) NOT NULL,
    result character varying(32),
    result_reason character varying(128),
    result_message text,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    configuration jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    cancelled_by uuid
);


--
-- Name: workflow_node_queue_items; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflow_node_queue_items (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    workflow_id uuid NOT NULL,
    node_id character varying(128) NOT NULL,
    root_event_id uuid,
    event_id uuid,
    created_at timestamp without time zone NOT NULL
);


--
-- Name: workflow_node_requests; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflow_node_requests (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    workflow_id uuid NOT NULL,
    execution_id uuid,
    state character varying(32) NOT NULL,
    type character varying(32) NOT NULL,
    spec jsonb NOT NULL,
    run_at timestamp without time zone NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    node_id character varying(128) NOT NULL
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
    updated_at timestamp without time zone NOT NULL,
    webhook_id uuid,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    "position" jsonb DEFAULT '{}'::jsonb NOT NULL,
    is_collapsed boolean DEFAULT false NOT NULL,
    parent_node_id character varying(128),
    deleted_at timestamp with time zone,
    app_installation_id uuid,
    state_reason character varying(255) DEFAULT NULL::character varying
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
    edges jsonb DEFAULT '[]'::jsonb NOT NULL,
    created_by uuid,
    deleted_at timestamp without time zone,
    nodes jsonb DEFAULT '[]'::jsonb NOT NULL,
    is_template boolean DEFAULT false NOT NULL
);


--
-- Name: casbin_rule id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.casbin_rule ALTER COLUMN id SET DEFAULT nextval('public.casbin_rule_id_seq'::regclass);


--
-- Name: account_password_auth account_password_auth_account_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_password_auth
    ADD CONSTRAINT account_password_auth_account_id_key UNIQUE (account_id);


--
-- Name: account_password_auth account_password_auth_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_password_auth
    ADD CONSTRAINT account_password_auth_pkey PRIMARY KEY (id);


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
-- Name: app_installation_requests app_installation_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installation_requests
    ADD CONSTRAINT app_installation_requests_pkey PRIMARY KEY (id);


--
-- Name: app_installation_secrets app_installation_secrets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installation_secrets
    ADD CONSTRAINT app_installation_secrets_pkey PRIMARY KEY (id);


--
-- Name: app_installation_subscriptions app_installation_subscription_installation_id_workflow_id_n_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installation_subscriptions
    ADD CONSTRAINT app_installation_subscription_installation_id_workflow_id_n_key UNIQUE (installation_id, workflow_id, node_id);


--
-- Name: app_installation_subscriptions app_installation_subscriptions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installation_subscriptions
    ADD CONSTRAINT app_installation_subscriptions_pkey PRIMARY KEY (id);


--
-- Name: app_installations app_installations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installations
    ADD CONSTRAINT app_installations_pkey PRIMARY KEY (id);


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
-- Name: canvas_memories canvas_memories_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.canvas_memories
    ADD CONSTRAINT canvas_memories_pkey PRIMARY KEY (id);


--
-- Name: casbin_rule casbin_rule_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.casbin_rule
    ADD CONSTRAINT casbin_rule_pkey PRIMARY KEY (id);


--
-- Name: data_migrations data_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.data_migrations
    ADD CONSTRAINT data_migrations_pkey PRIMARY KEY (version);


--
-- Name: email_settings email_settings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_settings
    ADD CONSTRAINT email_settings_pkey PRIMARY KEY (id);


--
-- Name: email_settings email_settings_provider_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_settings
    ADD CONSTRAINT email_settings_provider_key UNIQUE (provider);


--
-- Name: group_metadata group_metadata_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_metadata
    ADD CONSTRAINT group_metadata_pkey PRIMARY KEY (id);


--
-- Name: installation_metadata installation_metadata_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.installation_metadata
    ADD CONSTRAINT installation_metadata_pkey PRIMARY KEY (id);


--
-- Name: organization_agent_settings organization_agent_settings_organization_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_agent_settings
    ADD CONSTRAINT organization_agent_settings_organization_id_key UNIQUE (organization_id);


--
-- Name: organization_agent_settings organization_agent_settings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_agent_settings
    ADD CONSTRAINT organization_agent_settings_pkey PRIMARY KEY (id);


--
-- Name: organization_invitations organization_invitations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_invitations
    ADD CONSTRAINT organization_invitations_pkey PRIMARY KEY (id);


--
-- Name: organization_invite_links organization_invite_links_organization_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_invite_links
    ADD CONSTRAINT organization_invite_links_organization_id_key UNIQUE (organization_id);


--
-- Name: organization_invite_links organization_invite_links_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_invite_links
    ADD CONSTRAINT organization_invite_links_pkey PRIMARY KEY (id);


--
-- Name: organization_invite_links organization_invite_links_token_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_invite_links
    ADD CONSTRAINT organization_invite_links_token_key UNIQUE (token);


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
-- Name: webhooks webhooks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhooks
    ADD CONSTRAINT webhooks_pkey PRIMARY KEY (id);


--
-- Name: workflow_events workflow_events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_events
    ADD CONSTRAINT workflow_events_pkey PRIMARY KEY (id);


--
-- Name: workflow_node_execution_kvs workflow_node_execution_kvs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_execution_kvs
    ADD CONSTRAINT workflow_node_execution_kvs_pkey PRIMARY KEY (id);


--
-- Name: workflow_node_requests workflow_node_execution_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_requests
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
-- Name: idx_account_password_auth_account_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_account_password_auth_account_id ON public.account_password_auth USING btree (account_id);


--
-- Name: idx_account_providers_account_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_account_providers_account_id ON public.account_providers USING btree (account_id);


--
-- Name: idx_account_providers_provider; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_account_providers_provider ON public.account_providers USING btree (provider);


--
-- Name: idx_app_installation_requests_installation_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_app_installation_requests_installation_id ON public.app_installation_requests USING btree (app_installation_id);


--
-- Name: idx_app_installation_requests_state_run_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_app_installation_requests_state_run_at ON public.app_installation_requests USING btree (state, run_at) WHERE ((state)::text = 'pending'::text);


--
-- Name: idx_app_installation_secrets_installation_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_app_installation_secrets_installation_id ON public.app_installation_secrets USING btree (installation_id);


--
-- Name: idx_app_installation_secrets_organization_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_app_installation_secrets_organization_id ON public.app_installation_secrets USING btree (organization_id);


--
-- Name: idx_app_installation_subscriptions_installation; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_app_installation_subscriptions_installation ON public.app_installation_subscriptions USING btree (installation_id);


--
-- Name: idx_app_installation_subscriptions_node; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_app_installation_subscriptions_node ON public.app_installation_subscriptions USING btree (workflow_id, node_id);


--
-- Name: idx_app_installation_subscriptions_workflow; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_app_installation_subscriptions_workflow ON public.app_installation_subscriptions USING btree (workflow_id);


--
-- Name: idx_app_installations_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_app_installations_deleted_at ON public.app_installations USING btree (deleted_at);


--
-- Name: idx_app_installations_org_name_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_app_installations_org_name_unique ON public.app_installations USING btree (organization_id, installation_name);


--
-- Name: idx_app_installations_organization_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_app_installations_organization_id ON public.app_installations USING btree (organization_id);


--
-- Name: idx_blueprints_organization_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_blueprints_organization_id ON public.blueprints USING btree (organization_id);


--
-- Name: idx_canvas_memories_canvas_namespace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_canvas_memories_canvas_namespace ON public.canvas_memories USING btree (canvas_id, namespace);


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
-- Name: idx_group_metadata_lookup; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_group_metadata_lookup ON public.group_metadata USING btree (group_name, domain_type, domain_id);


--
-- Name: idx_node_requests_state_run_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_node_requests_state_run_at ON public.workflow_node_requests USING btree (state, run_at) WHERE ((state)::text = 'pending'::text);


--
-- Name: idx_organization_agent_settings_organization_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_organization_agent_settings_organization_id ON public.organization_agent_settings USING btree (organization_id);


--
-- Name: idx_organizations_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_organizations_deleted_at ON public.organizations USING btree (deleted_at);


--
-- Name: idx_role_metadata_lookup; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_role_metadata_lookup ON public.role_metadata USING btree (role_name, domain_type, domain_id);


--
-- Name: idx_webhooks_app_installation_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_webhooks_app_installation_id ON public.webhooks USING btree (app_installation_id);


--
-- Name: idx_webhooks_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_webhooks_deleted_at ON public.webhooks USING btree (deleted_at);


--
-- Name: idx_workflow_events_execution_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_events_execution_id ON public.workflow_events USING btree (execution_id);


--
-- Name: idx_workflow_events_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_events_state ON public.workflow_events USING btree (state);


--
-- Name: idx_workflow_events_workflow_node_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_events_workflow_node_id ON public.workflow_events USING btree (workflow_id, node_id);


--
-- Name: idx_workflow_node_execution_kvs_ekv; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_execution_kvs_ekv ON public.workflow_node_execution_kvs USING btree (execution_id, key, value);


--
-- Name: idx_workflow_node_execution_kvs_workflow_node_key_value; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_execution_kvs_workflow_node_key_value ON public.workflow_node_execution_kvs USING btree (workflow_id, node_id, key, value);


--
-- Name: idx_workflow_node_executions_event_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_executions_event_id ON public.workflow_node_executions USING btree (event_id);


--
-- Name: idx_workflow_node_executions_parent_execution_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_executions_parent_execution_id ON public.workflow_node_executions USING btree (parent_execution_id);


--
-- Name: idx_workflow_node_executions_parent_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_executions_parent_state ON public.workflow_node_executions USING btree (parent_execution_id, state);


--
-- Name: idx_workflow_node_executions_previous_execution_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_executions_previous_execution_id ON public.workflow_node_executions USING btree (previous_execution_id);


--
-- Name: idx_workflow_node_executions_root_event_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_executions_root_event_id ON public.workflow_node_executions USING btree (root_event_id);


--
-- Name: idx_workflow_node_executions_state_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_executions_state_created_at ON public.workflow_node_executions USING btree (state, created_at DESC);


--
-- Name: idx_workflow_node_executions_workflow_node_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_executions_workflow_node_id ON public.workflow_node_executions USING btree (workflow_id, node_id);


--
-- Name: idx_workflow_node_installation_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_installation_id ON public.workflow_nodes USING btree (app_installation_id);


--
-- Name: idx_workflow_node_queue_items_root_event_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_queue_items_root_event_id ON public.workflow_node_queue_items USING btree (root_event_id);


--
-- Name: idx_workflow_node_requests_execution_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_requests_execution_id ON public.workflow_node_requests USING btree (execution_id);


--
-- Name: idx_workflow_nodes_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_nodes_deleted_at ON public.workflow_nodes USING btree (deleted_at);


--
-- Name: idx_workflow_nodes_parent; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_nodes_parent ON public.workflow_nodes USING btree (workflow_id, parent_node_id);


--
-- Name: idx_workflow_nodes_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_nodes_state ON public.workflow_nodes USING btree (state);


--
-- Name: idx_workflows_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflows_deleted_at ON public.workflows USING btree (deleted_at);


--
-- Name: idx_workflows_is_template; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflows_is_template ON public.workflows USING btree (is_template);


--
-- Name: idx_workflows_organization_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflows_organization_id ON public.workflows USING btree (organization_id);


--
-- Name: unique_human_user_in_organization; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX unique_human_user_in_organization ON public.users USING btree (organization_id, account_id, email) WHERE ((type)::text = 'human'::text);


--
-- Name: unique_service_account_in_organization; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX unique_service_account_in_organization ON public.users USING btree (organization_id, name) WHERE ((type)::text = 'service_account'::text);


--
-- Name: account_password_auth account_password_auth_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_password_auth
    ADD CONSTRAINT account_password_auth_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id) ON DELETE CASCADE;


--
-- Name: account_providers account_providers_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_providers
    ADD CONSTRAINT account_providers_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id);


--
-- Name: app_installation_requests app_installation_requests_app_installation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installation_requests
    ADD CONSTRAINT app_installation_requests_app_installation_id_fkey FOREIGN KEY (app_installation_id) REFERENCES public.app_installations(id) ON DELETE CASCADE;


--
-- Name: app_installation_secrets app_installation_secrets_installation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installation_secrets
    ADD CONSTRAINT app_installation_secrets_installation_id_fkey FOREIGN KEY (installation_id) REFERENCES public.app_installations(id) ON DELETE CASCADE;


--
-- Name: app_installation_secrets app_installation_secrets_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installation_secrets
    ADD CONSTRAINT app_installation_secrets_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: app_installation_subscriptions app_installation_subscriptions_installation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installation_subscriptions
    ADD CONSTRAINT app_installation_subscriptions_installation_id_fkey FOREIGN KEY (installation_id) REFERENCES public.app_installations(id) ON DELETE CASCADE;


--
-- Name: app_installation_subscriptions app_installation_subscriptions_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installation_subscriptions
    ADD CONSTRAINT app_installation_subscriptions_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id) ON DELETE CASCADE;


--
-- Name: app_installation_subscriptions app_installation_subscriptions_workflow_id_node_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installation_subscriptions
    ADD CONSTRAINT app_installation_subscriptions_workflow_id_node_id_fkey FOREIGN KEY (workflow_id, node_id) REFERENCES public.workflow_nodes(workflow_id, node_id) ON DELETE CASCADE;


--
-- Name: app_installations app_installations_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installations
    ADD CONSTRAINT app_installations_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: canvas_memories canvas_memories_canvas_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.canvas_memories
    ADD CONSTRAINT canvas_memories_canvas_id_fkey FOREIGN KEY (canvas_id) REFERENCES public.workflows(id) ON DELETE CASCADE;


--
-- Name: workflow_node_execution_kvs fk_wnek_workflow; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_execution_kvs
    ADD CONSTRAINT fk_wnek_workflow FOREIGN KEY (workflow_id) REFERENCES public.workflows(id);


--
-- Name: workflow_node_execution_kvs fk_wnek_workflow_node; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_execution_kvs
    ADD CONSTRAINT fk_wnek_workflow_node FOREIGN KEY (workflow_id, node_id) REFERENCES public.workflow_nodes(workflow_id, node_id);


--
-- Name: workflow_events fk_workflow_events_workflow_node; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_events
    ADD CONSTRAINT fk_workflow_events_workflow_node FOREIGN KEY (workflow_id, node_id) REFERENCES public.workflow_nodes(workflow_id, node_id);


--
-- Name: workflow_node_executions fk_workflow_node_executions_workflow_node; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_executions
    ADD CONSTRAINT fk_workflow_node_executions_workflow_node FOREIGN KEY (workflow_id, node_id) REFERENCES public.workflow_nodes(workflow_id, node_id);


--
-- Name: workflow_node_queue_items fk_workflow_node_queue_items_workflow_node; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_queue_items
    ADD CONSTRAINT fk_workflow_node_queue_items_workflow_node FOREIGN KEY (workflow_id, node_id) REFERENCES public.workflow_nodes(workflow_id, node_id);


--
-- Name: workflow_node_requests fk_workflow_node_requests_workflow_node; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_requests
    ADD CONSTRAINT fk_workflow_node_requests_workflow_node FOREIGN KEY (workflow_id, node_id) REFERENCES public.workflow_nodes(workflow_id, node_id);


--
-- Name: workflow_nodes fk_workflow_nodes_parent; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_nodes
    ADD CONSTRAINT fk_workflow_nodes_parent FOREIGN KEY (workflow_id, parent_node_id) REFERENCES public.workflow_nodes(workflow_id, node_id) ON DELETE CASCADE;


--
-- Name: organization_agent_settings organization_agent_settings_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_agent_settings
    ADD CONSTRAINT organization_agent_settings_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: organization_agent_settings organization_agent_settings_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_agent_settings
    ADD CONSTRAINT organization_agent_settings_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: organization_invitations organization_invitations_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_invitations
    ADD CONSTRAINT organization_invitations_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: organization_invite_links organization_invite_links_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_invite_links
    ADD CONSTRAINT organization_invite_links_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: users users_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id);


--
-- Name: users users_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: users users_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id);


--
-- Name: webhooks webhooks_app_installation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhooks
    ADD CONSTRAINT webhooks_app_installation_id_fkey FOREIGN KEY (app_installation_id) REFERENCES public.app_installations(id);


--
-- Name: workflow_events workflow_events_execution_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_events
    ADD CONSTRAINT workflow_events_execution_id_fkey FOREIGN KEY (execution_id) REFERENCES public.workflow_node_executions(id) ON DELETE CASCADE;


--
-- Name: workflow_events workflow_events_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_events
    ADD CONSTRAINT workflow_events_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id);


--
-- Name: workflow_node_execution_kvs workflow_node_execution_kvs_execution_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_execution_kvs
    ADD CONSTRAINT workflow_node_execution_kvs_execution_id_fkey FOREIGN KEY (execution_id) REFERENCES public.workflow_node_executions(id) ON DELETE CASCADE;


--
-- Name: workflow_node_executions workflow_node_executions_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_executions
    ADD CONSTRAINT workflow_node_executions_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.workflow_events(id) ON DELETE SET NULL;


--
-- Name: workflow_node_executions workflow_node_executions_parent_execution_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_executions
    ADD CONSTRAINT workflow_node_executions_parent_execution_id_fkey FOREIGN KEY (parent_execution_id) REFERENCES public.workflow_node_executions(id) ON DELETE CASCADE;


--
-- Name: workflow_node_executions workflow_node_executions_previous_execution_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_executions
    ADD CONSTRAINT workflow_node_executions_previous_execution_id_fkey FOREIGN KEY (previous_execution_id) REFERENCES public.workflow_node_executions(id) ON DELETE SET NULL;


--
-- Name: workflow_node_executions workflow_node_executions_root_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_executions
    ADD CONSTRAINT workflow_node_executions_root_event_id_fkey FOREIGN KEY (root_event_id) REFERENCES public.workflow_events(id) ON DELETE SET NULL;


--
-- Name: workflow_node_executions workflow_node_executions_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_executions
    ADD CONSTRAINT workflow_node_executions_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id);


--
-- Name: workflow_node_queue_items workflow_node_queue_items_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_queue_items
    ADD CONSTRAINT workflow_node_queue_items_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.workflow_events(id) ON DELETE SET NULL;


--
-- Name: workflow_node_queue_items workflow_node_queue_items_root_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_queue_items
    ADD CONSTRAINT workflow_node_queue_items_root_event_id_fkey FOREIGN KEY (root_event_id) REFERENCES public.workflow_events(id) ON DELETE SET NULL;


--
-- Name: workflow_node_queue_items workflow_node_queue_items_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_queue_items
    ADD CONSTRAINT workflow_node_queue_items_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id);


--
-- Name: workflow_node_requests workflow_node_requests_execution_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_requests
    ADD CONSTRAINT workflow_node_requests_execution_id_fkey FOREIGN KEY (execution_id) REFERENCES public.workflow_node_executions(id) ON DELETE CASCADE;


--
-- Name: workflow_node_requests workflow_node_requests_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_requests
    ADD CONSTRAINT workflow_node_requests_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id);


--
-- Name: workflow_nodes workflow_nodes_app_installation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_nodes
    ADD CONSTRAINT workflow_nodes_app_installation_id_fkey FOREIGN KEY (app_installation_id) REFERENCES public.app_installations(id) ON DELETE SET NULL;


--
-- Name: workflow_nodes workflow_nodes_webhook_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_nodes
    ADD CONSTRAINT workflow_nodes_webhook_id_fkey FOREIGN KEY (webhook_id) REFERENCES public.webhooks(id) ON DELETE SET NULL;


--
-- Name: workflow_nodes workflow_nodes_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_nodes
    ADD CONSTRAINT workflow_nodes_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id);


--
-- PostgreSQL database dump complete
--

\unrestrict abcdef123

--
-- PostgreSQL database dump
--

\restrict abcdef123

-- Dumped from database version 17.5 (Debian 17.5-1.pgdg130+1)
-- Dumped by pg_dump version 17.8 (Ubuntu 17.8-1.pgdg22.04+1)

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
20260227135919	f
\.


--
-- PostgreSQL database dump complete
--

\unrestrict abcdef123

--
-- PostgreSQL database dump
--

\restrict abcdef123

-- Dumped from database version 17.5 (Debian 17.5-1.pgdg130+1)
-- Dumped by pg_dump version 17.8 (Ubuntu 17.8-1.pgdg22.04+1)

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
-- Data for Name: data_migrations; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.data_migrations (version, dirty) FROM stdin;
20260202201226	f
\.


--
-- PostgreSQL database dump complete
--

\unrestrict abcdef123

