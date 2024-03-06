use std::cell::RefCell;

use ic_stable_structures::{
    memory_manager::{MemoryId, MemoryManager, VirtualMemory},
    DefaultMemoryImpl, StableBTreeMap,
};

use structs::Event;

pub mod methods;
pub mod structs;

pub type Memory = VirtualMemory<DefaultMemoryImpl>;
pub type StorageRef<K, V> = RefCell<StableBTreeMap<K, V, Memory>>;
type MemManagerStore = RefCell<MemoryManager<DefaultMemoryImpl>>;

thread_local! {
    pub static MEMORY_MANAGER: MemManagerStore =
    RefCell::new(MemoryManager::init(DefaultMemoryImpl::default()));

    pub static EVENTS: StorageRef<String, Event> = RefCell::new(
        StableBTreeMap::init(MEMORY_MANAGER.with(|p| p.borrow().get(MemoryId::new(0))))
    );
}
