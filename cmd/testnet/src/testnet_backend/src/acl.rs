

use std::{cell::{Ref, RefCell}, rc::Rc};

use candid::Principal;
use ic_cdk::caller;

use base64::{engine::general_purpose::STANDARD as BASE64_STANDARD, Engine};

use crate::{PERMISSIONS,owner::OWNER};

use ic_cdk_macros::{query, update,init};
use ic_stable_structures::{
    memory_manager::{MemoryId, MemoryManager, VirtualMemory},
    DefaultMemoryImpl, StableBTreeMap,
};






fn string_to_principal(s: String) -> Result<Principal, String> {
    match BASE64_STANDARD.decode(s) {
        Ok(bytes) => {
            let principal = Principal::self_authenticating(&bytes);
            Ok(principal)
        },
        Err(e) => Err(format!("Failed to decode base64: {}", e)),
    }
}

pub fn is_user() -> Result<(), String> {
    let caller = caller();
    let permissions = PERMISSIONS.with(|p| p.borrow().contains_key(&caller));
    
    if permissions {
        Ok(())
    } else {
        Err("Caller is not authorized.".to_string())
    }
}


pub fn is_owner() -> Result<(), String> {
    let caller = caller();
    let permissions = PERMISSIONS.with(|p| p.borrow().get(&caller));
    
    if let Some(permission) = permissions {
        if permission == true {
            Ok(())
        } else {
            Err("Caller is not the owner.".to_string())
        }
    } else {
        Err("Caller is not authorized.".to_string())
    }
}


pub fn add_user_acl(pub_key : String,perm : bool) -> String {
    let principal_res = string_to_principal(pub_key);
    if let Ok(principal) = principal_res {
        PERMISSIONS.with(|p| p.borrow_mut().insert(principal, perm));
        "success".to_string()
    } else {
        principal_res.err().unwrap()
    }
}


pub fn remove_user_acl(pub_key : String) -> String{
    

    let principal_res = string_to_principal(pub_key);
    if let Ok(principal) = principal_res {
        PERMISSIONS.with(|p| p.borrow_mut().remove(&principal));
        "success".to_string()
    } else {
        principal_res.err().unwrap()
    }
}



pub fn get_permission_acl() -> String {
    let principal: Principal 
    = caller();
    let permissions = PERMISSIONS.with(|p| p.borrow().get(&principal));
    if let Some(permission) = permissions {
        if permission == true {
            "Owner".to_string()
        } else {
            "User".to_string()
        }
    } else {
        "Unauthorized/Error".to_string()
    }
}
#[init]
pub fn init() {
    let principal_res = string_to_principal(OWNER.to_string());
    if let Ok(principal) = principal_res {
        if !PERMISSIONS.with(|permissions| {
            permissions.borrow().contains_key(&principal)
        }) {
            PERMISSIONS.with(|p| p.borrow_mut().insert(principal, true));
            ic_cdk::println!("Initialized");
        } else {
            ic_cdk::println!("Permissions already exist");
        }
    } else {
        ic_cdk::println!("{}",principal_res.err().unwrap());
    }
}
