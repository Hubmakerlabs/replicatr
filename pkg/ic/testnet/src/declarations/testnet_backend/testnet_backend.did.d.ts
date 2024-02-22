import type { Principal } from '@dfinity/principal';
import type { ActorMethod } from '@dfinity/agent';
import type { IDL } from '@dfinity/candid';

export interface Record { 'id' : bigint, 'content' : string }
export interface _SERVICE {
  'get_record' : ActorMethod<[bigint], [] | [Record]>,
  'save_record' : ActorMethod<[bigint, string], string>,
}
export declare const idlFactory: IDL.InterfaceFactory;
export declare const init: ({ IDL }: { IDL: IDL }) => IDL.Type[];
