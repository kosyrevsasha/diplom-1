CREATE TABLE IF NOT EXISTS users (
    id bigserial PRIMARY KEY,
    login character varying NOT NULL,
    password character varying NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS user_login_idx ON users (login);

CREATE TABLE IF NOT EXISTS public.orders (
    id character varying NOT NULL,
    status character varying,
    accrual real DEFAULT 0,
    uploaded_at timestamp without time zone DEFAULT now(),
    withdrawal boolean DEFAULT false,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS public.user_orders (
    user_id integer NOT NULL,
    order_id character varying NOT NULL,
    CONSTRAINT fk_user_id FOREIGN KEY (user_id)
    REFERENCES public.users (id) MATCH SIMPLE,
    CONSTRAINT fk_order_id FOREIGN KEY (order_id)
    REFERENCES public.orders (id) MATCH SIMPLE
);

-- ALTER TABLE IF EXISTS public.users OWNER to go;
-- ALTER TABLE IF EXISTS public.orders OWNER to go;
-- ALTER TABLE IF EXISTS public.user_orders OWNER to go;
