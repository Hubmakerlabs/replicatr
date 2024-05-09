



use candid::Principal;
use ic_cdk::caller;

use base64::{engine::general_purpose::STANDARD as BASE64_STANDARD, Engine};

use crate::{PERMISSIONS,owner::OWNER};

use ic_cdk_macros::init;







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


pub fn add_user_acl(pub_key : String,perm : bool) -> Result<(),String> {
    let principal_res = string_to_principal(pub_key);
    match principal_res {
        Ok(principal) => {
            PERMISSIONS.with(|p| p.borrow_mut().insert(principal, perm));
            Ok(())
        },
        Err(e) => Err(e),
    }
}


pub fn remove_user_acl(pub_key : String) -> Result<(),String>{
    

    let principal_res = string_to_principal(pub_key);
    match principal_res {
        Ok(principal) => {
            PERMISSIONS.with(|p| p.borrow_mut().remove(&principal));
            Ok(())
        },
        Err(e) => Err(e),
    }
}



pub fn get_permission_acl() -> Result<String,String> {
    let principal: Principal 
    = caller();
    let permissions = PERMISSIONS.with(|p| p.borrow().get(&principal));
    match permissions {
        Some(permission) => if permission == true {
            Ok("Owner".to_string())
        } else {
            Ok("User".to_string())
        },
        None => Err("Unauthorized".to_string()),
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
