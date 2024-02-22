export const idlFactory = ({ IDL }) => {
  const Record = IDL.Record({ 'id' : IDL.Nat64, 'content' : IDL.Text });
  return IDL.Service({
    'get_record' : IDL.Func([IDL.Nat64], [IDL.Opt(Record)], ['query']),
    'save_record' : IDL.Func([IDL.Nat64, IDL.Text], [IDL.Text], []),
  });
};
export const init = ({ IDL }) => { return []; };
