export type ConnectFormValues = {
  host: string;
};

export const connectFormDefaults = (host = ''): ConnectFormValues => ({
  host,
});
