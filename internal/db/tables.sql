CREATE SEQUENCE public.users_id_seq
    START WITH 1
    NO MINVALUE
    NO MAXVALUE
;

CREATE TABLE public.users (
    id int8 PRIMARY KEY DEFAULT nextval('public.users_id_seq'),

    login    text NOT NULL,
    password text NOT NULL,

    create_ts int8 NOT NULL, -- unix microseconds timestamp

    UNIQUE (login)
);

CREATE TABLE public.sessions (
    token   text PRIMARY KEY,
    user_id int8 NOT NULL,

    create_ts int8 NOT NULL, -- unix microseconds timestamp
    expire_ts int8 NOT NULL, -- unix microseconds timestamp

    FOREIGN KEY (user_id) REFERENCES public.users(id)
);

CREATE SEQUENCE public.tasks_id_seq
    START WITH 1
    NO MINVALUE
    NO MAXVALUE
;

CREATE TABLE public.tasks (
    id      int8 PRIMARY KEY DEFAULT nextval('public.tasks_id_seq'),
    user_id int8 NOT NULL,

    state        int4 NOT NULL,
    importance   int4,
    progress     int4,
    max_progress int4
    deadline_ts  int8,          -- unix microseconds timestamp
    title        text NOT NULL,
    description  text,

    create_ts int8 NOT NULL, -- unix microseconds timestamp
    update_ts int8,          -- unix microseconds timestamp

    FOREIGN KEY (user_id) REFERENCES public.users(id)
);

COMMENT ON COLUMN public.tasks.state IS
'Determines completeness and visibility to user.

0 - active
1 - done
2 - canceled
3 - expired
';

COMMENT ON COLUMN public.tasks.max_progress IS
'Maximum numerical progress for tasks which have progress indicator.

Equals NULL for tasks which do not have progress.
';

COMMENT ON COLUMN public.tasks.progress IS
'Current numerical progress for tasks which have progress indicator.

Equals NULL for tasks which do not have progress.
';

CREATE SEQUENCE public.tags_id_seq
    START WITH 1
    NO MINVALUE
    NO MAXVALUE
;

CREATE TABLE public.tags (
    id int8 PRIMARY KEY DEFAULT nextval('public.tags_id_seq'),

    name text NOT NULL,

    UNIQUE (name) 
);

CREATE TABLE public.task_tags (
    task_id int8 NOT NULL,
    tag_id  int8 NOT NULL,

    PRIMARY KEY (task_id, tag_id),

    FOREIGN KEY (task_id) REFERENCES public.tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES public.tags(id) ON DELETE CASCADE
);
