export type AliasFormValues = {
  alias: string;
  filter: string;
  index: string;
  index_routing: string;
  search_routing: string;
};

export const aliasFormDefaults: AliasFormValues = {
  alias: '',
  filter: '',
  index: '',
  index_routing: '',
  search_routing: '',
};
