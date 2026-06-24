export type ConnectFormValues = {
  host: string;
};

export type AuthFormValues = {
  password: string;
  username: string;
};

export const connectFormDefaults = (host = ''): ConnectFormValues => ({
  host,
});

export const authFormDefaults: AuthFormValues = {
  password: '',
  username: '',
};
