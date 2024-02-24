import type { Principal } from '@dfinity/principal';
import type { ActorMethod } from '@dfinity/agent';
import type { IDL } from '@dfinity/candid';

export interface Event {
  'id' : string,
  'sig' : string,
  'content' : string,
  'kind' : number,
  'tags' : Array<Array<string>>,
  'pubkey' : string,
  'created_at' : bigint,
}
export interface Filter {
  'ids' : Array<string>,
  'tags' : Array<KeyValuePair>,
  'search' : string,
  'limit' : bigint,
  'since' : bigint,
  'authors' : Array<string>,
  'until' : bigint,
  'kinds' : Uint16Array | number[],
}
export interface KeyValuePair { 'key' : string, 'value' : Array<string> }
export interface _SERVICE {
  'get_events' : ActorMethod<[Filter], Array<Event>>,
  'save_event' : ActorMethod<[Event], string>,
}
export declare const idlFactory: IDL.InterfaceFactory;
export declare const init: ({ IDL }: { IDL: IDL }) => IDL.Type[];
