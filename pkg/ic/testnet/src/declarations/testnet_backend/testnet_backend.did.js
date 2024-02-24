export const idlFactory = ({ IDL }) => {
  const KeyValuePair = IDL.Record({
    'key' : IDL.Text,
    'value' : IDL.Vec(IDL.Text),
  });
  const Filter = IDL.Record({
    'ids' : IDL.Vec(IDL.Text),
    'tags' : IDL.Vec(KeyValuePair),
    'search' : IDL.Text,
    'limit' : IDL.Int,
    'since' : IDL.Int,
    'authors' : IDL.Vec(IDL.Text),
    'until' : IDL.Int,
    'kinds' : IDL.Vec(IDL.Nat16),
  });
  const Event = IDL.Record({
    'id' : IDL.Text,
    'sig' : IDL.Text,
    'content' : IDL.Text,
    'kind' : IDL.Nat16,
    'tags' : IDL.Vec(IDL.Vec(IDL.Text)),
    'pubkey' : IDL.Text,
    'created_at' : IDL.Int,
  });
  return IDL.Service({
    'get_events' : IDL.Func([Filter], [IDL.Vec(Event)], ['query']),
    'save_event' : IDL.Func([Event], [IDL.Text], []),
  });
};
export const init = ({ IDL }) => { return []; };
